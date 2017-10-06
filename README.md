# E.V.E.

[![GoDoc](https://godoc.org/github.com/rvflash/eve?status.svg)](https://godoc.org/github.com/rvflash/eve)
[![Build Status](https://img.shields.io/travis/rvflash/eve.svg)](https://travis-ci.org/rvflash/eve)
[![Code Coverage](https://img.shields.io/codecov/c/github/rvflash/eve.svg)](http://codecov.io/github/rvflash/eve?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/rvflash/eve)](https://goreportcard.com/report/github.com/rvflash/eve)


E.V.E. is a distributed environment variables management tool based
on a friendly user interface named Environment Variables Editor. 
It bases on :

* A HTTP web interface to manage server nodes, projects, environments and variables.
* One or more RPC servers used to store the deployed variables values. 
* A library to retrieve environment variables from various handlers.


## Installation

`eve` requires Go 1.8 or later.

```bash
$ go get -u github.com/rvflash/eve
```


### Quick start

#### Launches the web interface

By default, the editor is available on the net address localhost:8080.
The database is a BoltDB file named eve.db created in the launch directory.
This interface is based on Bootstrap v4 and jQuery 3.2 to propose a simple interface.

```bash
cd $GOPATH/src/github.com/rvflash/eve/server/http/
go build && ./http
```

You can change its behavior by using the command flags:

```bash
./http --help
Usage of ./http:
  -dsn string
    	database's filepath (default "eve.db")
  -host string
    	host addr to listen on
  -port int
    	service port (default 8080)
``` 


#### Launches one instance of RPC cache where ever who want.

By default, the RPC server is available on the port 9090.

```bash
cd $GOPATH/src/github.com/rvflash/eve/server/tcp/
go build && ./tcp
```

You can change it with :

```bash
./tcp --help
Usage of ./tcp:
  -host string
    	host addr to listen on
  -port int
    	service port (default 9090) 
```

Now, you can open your favorite browser and go to http://localhost:8080 to create your first project.
Its name: Alpha.

After, you can if necessary add until two environments to vary the variable's values in case of these. 
By example, you can create one environment named `Env`with `dev`, `qa` or `prod` as values.
For each variable afterwards, you can vary the value. 

To finalize your discovery, you should add the net address of the RPC server as your first node cache.
Then, deploy the variables for the environment of your choice in this remote or local cache.


### Usage

Now you can use the E.V.E. library to access to your environment variables.
You can schedule as you want the client to use. 

By default, the handler is defined to lookup in its local cache, then in the OS environment and
finally in the cache servers add with the New method. 

```go
// Import the E.V.E. library.
import "github.com/rvflash/eve"

// ...

// Use the net address of the RPC cache started before. 
caches, err := eve.Servers(":9090")
if err != nil {
    fmt.Println(err)
    return
}

// Launch the E.V.E. handler.
// The names of project, environment or variable are not case sensitive.
// Moreover, dash will be internally replace with an underscore. 
// Start by setting the name of the project: alpha.
vars := eve.New("alpha", caches...)

// Alpha is defined to have one environment.
// Here we set the current environment value.
if err := vars.Envs("qa"); err != nil {
    fmt.Println(err)
    return
}

// Now, we suppose to have create 3 variables named:
// enabled, keyword and value.
// With this configuration, E.V.E. will try to lookup the following variables:
// ALPHA_QA_ENABLED, ALPHA_QA_KEYWORD and ALPHA_QA_VALUE.
if vars.MustBool("enabled") {
    str, err := vars.String("keyword")
    if err != nil {
    	fmt.Println(err)
        return
    }
    fmt.Print(str)
}
if data, ok := vars.Lookup("value"); ok {
    fmt.Printf(": %d", data.(int))
}
// Output: rv: 42
```


## More features

* You can use your own client to supply the environment variables by implementing the client.Getter interface.
* More client interfaces can be used: one to check the client's availability to disable the internal cache recycle.
* Another interface named client.Assert can be used to realize assertion on data of your client. 