# Slammer

Slammer is a simple utility using Moldova, for load testing a database. You can give it a template SQL query, most likely an INSERT
statement, and use that to generate a large volume of traffic, each request having a different set of values placed into it. In this way,
the Slammer makes an excellent tool for massively loading fake data into a database for load testing.

Slammer comes as a binary executable, and accepts lines of SQL in via STDIN, either through a pipe or through io redirection

## Supported Databases

Since Slammer does not generate SQL, it is agnostic to what kind of database is being used.

That said, currently it only has support for databases that are wire-compatible with either
postgres or mysql. If you know of another sql/db driver you'd like to see supported, please
open an issue or a PR.

## Works great with Moldova

Slammer was originally designed to receive input from [Moldova](https://github.com/StabbyCutyou/moldova), a lightweight
template generation utility which is good for generating random data. By using Moldova, you can generate a stream
of random insert or select statements, which can be fed to the STDIN of Slammer.

And if you're interested in using them together, checkout the [Moldovan Slammer](http://github.com/StabbyCutyou/moldovan_slammer), a quick helper repo to
demonstrate using them together

## Slammer Command

To use the command, first install

```bash
go get github.com/StabbyCutyou/slammer
```

The command accepts several arguments:

* p - How long of a pause to take between each statement. The default is 1 second, and it can be any valid value that parses to a time.Duration.
* db - Which driver to load. Currently supports "mysql" and "postgres". The default is "mysql".
* c - The connection string to the database.
* w - The size of the worker pool. This defaults to 1, and cannot be lower than 1.
* d - Debug mode. Enable this to see errors logged to STDERR as they happen

## Example

```bash
slammer -c "root@tcp(127.0.0.1:3306)/my_db" -p 200us -w 2 < mysqlfile
```

This would provide sample output like the following:

```
Slammer Status:
Queries to run: 200
---- Worker #0 ----
  Started at 2016-01-29 22:00:44 , Ended at 2016-01-29 22:00:51, took ...
  Total work: ..., Percentage work: ... , Average work per second: ...
  Total errors: ... , Percentage errors: ..., Average errors per second: ...
---- Worker #1 ----
  Started at 2016-01-29 22:00:44 , Ended at 2016-01-29 22:00:51, took ...
  Total work: ..., Percentage work: ..., Average work per second: ...
  Total errors: ... , Percentage errors: ..., Average errors per second: ...
```

## Roadmap

Support for more databases

Improve the summary at the end to include overall statistics

"Interactive" mode, with sparklines shown in the terminal to demonstrate latency
and throughput with real time visualizations

# License

Apache v2 - See LICENSE
