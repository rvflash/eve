# E.V.E.

[![GoDoc](https://godoc.org/github.com/rvflash/eve?status.svg)](https://godoc.org/github.com/rvflash/eve)
[![Build Status](https://img.shields.io/travis/rvflash/eve.svg)](https://travis-ci.org/rvflash/eve)
[![Code Coverage](https://img.shields.io/codecov/c/github/rvflash/eve.svg)](http://codecov.io/github/rvflash/eve?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/rvflash/eve)](https://goreportcard.com/report/github.com/rvflash/eve)


E.V.E. is a environment variables management tool based on
a friendly user interface named Environment Variables Editor. 

* A HTTP web interface to manage server nodes, projects, environments and variables.
* One or more RPC servers used to store the deployed variables values. 
* A library to retrieve environment variables from various handlers and schedule the get order.


## Installation

`eve` requires Go 1.5 or later. (os.LookupEnv is required)
It uses go dep to manage dependencies.

```bash
$ go get -u github.com/rvflash/eve
$ cd $GOPATH/src/github.com/rvflash/eve
$ dep ensure
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

E.V.E. exposes all environment variables behind /vars.
Thereby, to deploy a new instance of cache with all managed variables loaded on starting, we can use the `from` option to specify this URL.

```bash
cd $GOPATH/src/github.com/rvflash/eve/server/tcp/
go build && ./tcp -from "http://localhost:8080/vars"
```

You can change it with :

```bash
./tcp --help
Usage of ./tcp:
  -from string
    	URL to fetch to get JSON data to use as default values
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
finally in the cache servers added with the New method. 


#### Uses the data getters

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

// Now, we suppose to have created 3 variables named: enabled, keyword and value.
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


##### Processes the struct's fields.

E.V.E. supports the use of struct tags to specify alternate name and required environment variables.

`eve` can be used to specify an alternate name and `required` with `true` as value, marks as mandatory the field.

E.V.E has automatic support for CamelCased structure fields.
In the following example, in the continuity of the previous sample, it searches for the variables named ALPHA_QA_OTHER_NAME and ALPHA_QA_REQUIRED_VAR.
If the last variable can not be found, as the field is tag as mandatory, the `Process` will return in error.

```go
// MyCnf is sample struct to feed.
type MyCnf struct {
    AliasVar    string  `eve:"OTHER_NAME"`
    RequiredVar int     `required:"true"`
}

// ...

var conf MyCnf
if err := vars.Process(&conf); err != nil {
    fmt.Println(err)
    return
}
```


##### Supported structure field types

* string
* int, int8, int16, int32, int64
* uint, uint8, uint16, uint32, uint64
* bool
* float32, float64
* time.Duration

Soon, E.V.E. will manage time.Time, slices and maps of any supported type. 


## More features

* You can use your own client to supply the environment variables by implementing the client.Getter interface.
* More client interfaces can be used: one to check the client's availability to disable the internal cache recycle.
* Another interface named client.Asserter can be used to realize assertion on data of your client.