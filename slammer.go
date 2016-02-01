package main

import (
	"bufio"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	// Load the drivers
	// MySQL
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type config struct {
	connString    string
	db            string
	pauseInterval time.Duration
	workers       int
	debugMode     bool
}

type result struct {
	start     time.Time
	end       time.Time
	dbTime    time.Duration
	workCount int
	errors    int
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open(cfg.db, cfg.connString)
	if err != nil {
		log.Fatal(err)
	}

	// Declare the channel we'll be using as a work queue
	inputChan := make(chan (string))
	// Declare the channel that will gather results
	outputChan := make(chan (result), cfg.workers)
	// Declare a waitgroup to help prevent log interleaving - I technically do not
	// need one, but without it, I find there are stray log messages creeping into
	// the final report. Setting sync() on STDOUT/ERR didn't seem to fix it.
	var wg sync.WaitGroup
	wg.Add(cfg.workers)

	// Start the pool of workers up, reading from the channel
	startWorkers(cfg.workers, inputChan, outputChan, db, &wg, cfg.pauseInterval, cfg.debugMode)

	// Warm up error and line so I can use error in the for loop with running into
	// a shadowing issue
	err = nil
	line := ""
	totalWorkCount := 0
	// Read from STDIN in the main thread
	input := bufio.NewReader(os.Stdin)
	for err != io.EOF {
		line, err = input.ReadString('\n')
		if err == nil {
			// Get rid of any unwanted stuff
			line = strings.TrimRight(line, "\r\n")
			// Push that onto the work queue
			inputChan <- line
			totalWorkCount++
		} else if cfg.debugMode {
			log.Println(err)
		}
	}

	// Close the channel, since it's done receiving input
	close(inputChan)
	// As I mentioned above, because workers wont finish at the same time, I need
	// to use a waitgroup, otherwise the output below gets potentially mixed in with
	// debug or error messages from the workers. The waitgroup semaphore prevents this
	// even though it probably looks redundant
	wg.Wait()
	// Collect all results, report them. This will block and wait until all results
	// are in
	fmt.Println("Slammer Status:")
	fmt.Printf("Queries to run: %d\n", totalWorkCount)
	for i := 0; i < cfg.workers; i++ {
		r := <-outputChan
		workerDuration := r.end.Sub(r.start)
		fmt.Printf("---- Worker #%d ----\n", i)
		fmt.Printf("  Started at %s , Ended at %s, Worker time %s, DB time %s\n", r.start.Format("2006-01-02 15:04:05"), r.end.Format("2006-01-02 15:04:05"), workerDuration.String(), r.dbTime)
		fmt.Printf("  Total work: %d, Percentage work: %f, Average work over DB time: %f\n", r.workCount, float64(r.workCount)/float64(totalWorkCount), float64(r.workCount)/float64(r.dbTime))
		fmt.Printf("  Total errors: %d , Percentage errors: %f, Average errors per second: %f\n", r.errors, float64(r.errors)/float64(r.workCount), float64(r.errors)/workerDuration.Seconds())
	}

	// Lets just be nice and tidy
	close(outputChan)
}

func startWorkers(count int, ic <-chan string, oc chan<- result, db *sql.DB, wg *sync.WaitGroup, pause time.Duration, debugMode bool) {
	// Start the pool of workers up, reading from the channel
	for i := 0; i < count; i++ {
		// register a signal chan for handling shutdown
		sc := make(chan os.Signal)
		signal.Notify(sc, os.Interrupt)
		// Pass in everything it needs
		go startWorker(i, ic, oc, sc, db, wg, pause, debugMode)
	}
}

func startWorker(workerNum int, ic <-chan string, oc chan<- result, sc <-chan os.Signal, db *sql.DB, done *sync.WaitGroup, pause time.Duration, debugMode bool) {
	// Prep the result object
	r := result{start: time.Now()}
	for line := range ic {
		// First thing is first - do a non blocking read from the signal channel, and
		// handle it if something came through the pipe
		select {
		case _ = <-sc:
			// UGH I ACTUALLY ALMOST USED A GOTO HERE BUT I JUST CANT DO IT
			// NO NO NO NO NO NO I WONT YOU CANT MAKE ME NO
			// I could put it into an anonymous function defer, though...
			r.end = time.Now()
			oc <- r
			done.Done()
			return
		default:
			// NOOP
		}
		t := time.Now()
		_, err := db.Exec(line)
		r.dbTime += time.Since(t)
		// TODO should this be after the err != nil? It counts towards work attempted
		// but not work completed.
		r.workCount++
		if err != nil {
			r.errors++
			if debugMode {
				log.Printf("Worker #%d: %s - %s", workerNum, line, err.Error())
			}
		} else {
			// Sleep for the configured amount of pause time between each call
			time.Sleep(pause)
		}
	}

	// Let everyone know we're done, and bail out
	r.end = time.Now()
	oc <- r
	done.Done()
}

func getConfig() (*config, error) {
	p := flag.String("p", "1s", "The time to pause between each call to the database")
	c := flag.String("c", "", "The connection string to use when connecting to the database")
	db := flag.String("db", "mysql", "The database driver to load. Defaults to mysql")
	w := flag.Int("w", 1, "The number of workers to use. A number greater than 1 will enable statements to be issued concurrently")
	d := flag.Bool("d", false, "Debug mode - turn this on to have errors printed to the terminal")
	flag.Parse()

	if *c == "" {
		return nil, errors.New("You must provide a connection string using the -c option")
	}
	pi, err := time.ParseDuration(*p)
	if err != nil {
		return nil, errors.New("You must provide a proper duration value with -p")
	}

	if *w <= 0 {
		return nil, errors.New("You must provide a worker count > 0 with -w")
	}

	return &config{db: *db, connString: *c, pauseInterval: pi, workers: *w, debugMode: *d}, nil
}
