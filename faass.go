package main

import (
    "encoding/json" 
    "io/ioutil"
    "os"
    "os/exec" 
    "reflect"
    "errors" 
    "flag"
    "net/http"
    "strings"
    "log"
    // "net"
    "time"
    "html"
    "fmt" 
    "sync"
)

// ----------------------------------------------- 

var GLOBAL_CONF_PATH string 
var GLOBAL_CONF Conf 

// ----------------------------------------------- 

const (
    ExitOk = iota           // toujours '0' 
    ExitUndefined           // toujours '1' 
    ExitConfCreateKo
    ExitConfLoadKo
)

// ----------------------------------------------- 

var (
    DebugLogger     *log.Logger
    InfoLogger      *log.Logger
    WarningLogger   *log.Logger
    ErrorLogger     *log.Logger
    PanicLogger     *log.Logger
)

// ----------------------------------------------- 

// func redirectionHandler(w http.ResponseWriter, r *http.Request) {
//     url := GLOBAL_DOMAIN + r.URL.Path[10:] // "/redirect/" = 10 signes 
//     http.Redirect(w, r, url, 301)
// } 

// ----------------------------------------------- 

type Conf struct {
    Containers Containers 
    Domain string `json:"domain"`
    UI string `json:"ui"`
    TmpDir string `json:"dirtmp"`
    Prefix string `json:"prefix"`
    Routes map[string]*Route `json:"routes"`
} 

func ConfImport( pathOption ...string ) bool { 
    path := GLOBAL_CONF_PATH 
    if len( pathOption ) < 0 {
        path = pathOption[0] 
    } 
    jsonFileInput, err := os.Open( path )
    if err != nil {
        log.Println( "ConfImport (open) :", err ) 
        return false
    }
    defer jsonFileInput.Close() 
    byteValue, err := ioutil.ReadAll(jsonFileInput)
    if err != nil {
        log.Println( "ConfImport (ioutil) :", err ) 
        return false
    }
    json.Unmarshal( byteValue, &GLOBAL_CONF ) 
    return true 
} 

func (c *Conf) GetParam( key string ) string {
    e := reflect.ValueOf( c ).Elem() 
    r := e.FieldByName( key ) 
    if r.IsValid() {
        return r.Interface().(string) 
    } 
    return ""
} 

func (c *Conf) GetRoute( key string ) (route *Route, err error) { 
    if route, ok := c.Routes[key]; ok {
        return route, nil
    } 
    return nil, errors.New( "unknow routes" ) 
}

func (c *Conf) Export( pathOption ...string ) bool {
    path := GLOBAL_CONF_PATH 
    if len( pathOption ) < 0 {
        path = pathOption[0] 
    } 
    v, err := json.Marshal( GLOBAL_CONF ) 
    if err != nil {
        log.Fatal( "export conf (Marshal) :", err ) 
        return false
    } 
    jsonFileOutput, err := os.Create( path ) 
    defer jsonFileOutput.Close() 
    if err != nil {
        log.Println( "export conf (open) :", err ) 
        return false
    } 
    _, err = jsonFileOutput.Write( v ) 
    if err != nil {
        log.Println( "export conf (write) :", err ) 
        return false
    } 
    return true 
} 

// ----------------------------------------------- 

func CreateLogger() {
    m := "!!! starting ; test log" 
    DebugLogger = log.New( os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile )
    DebugLogger.Println( m ) 
    InfoLogger = log.New( os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile )
    InfoLogger.Println( m )
    WarningLogger = log.New( os.Stderr, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile )
    WarningLogger.Println( m )
    ErrorLogger = log.New( os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile )
    ErrorLogger.Println( m )
    PanicLogger = log.New( os.Stderr, "PANIC: ", log.Ldate|log.Ltime|log.Lshortfile )
    PanicLogger.Println( m )    
}

func CreateEnv() bool {
    uiTmpDir := "./content" 
    pathTmpDir := "./tmp" 
    GLOBAL_CONF = Conf { 
        Containers: Containers{}, 
        Domain: "https://localhost", 
        UI: uiTmpDir, 
        TmpDir: pathTmpDir, 
        Prefix: "lambda", 
    } 
    if ! GLOBAL_CONF.Export() {
        log.Println( "Unable to create environment : conf" ) 
        return false 
    }
    if err := os.Mkdir( uiTmpDir, os.ModePerm ); err != nil {
        log.Println( "Unable to create environment : content dir (", err, "); pass" ) 
        return false 
    }
    if err := os.Mkdir( pathTmpDir, os.ModePerm ); err != nil {
        log.Println( "Unable to create environment : tmp dir (", err, "); pass" ) 
        return false 
    } 
    return true 
}

func StartEnv() {
    confPath := flag.String( "conf", "./conf.json", "path to conf (JSON ; string)" )  
    prepareEnv := flag.Bool( "prepare", false, "create environment (conf+dir ; bool)" ) 
    flag.Parse() 
    GLOBAL_CONF_PATH = string( *confPath ) 
    if *prepareEnv { 
        if CreateEnv() { 
            os.Exit( ExitOk )
        } else { 
            os.Exit( ExitConfCreateKo ) 
        } 
    } 
    if !ConfImport() { 
        log.Println( "Unable to load configuration" ) 
        os.Exit( ExitConfLoadKo )
    } 
} 

// ----------------------------------------------- 

// func ClientContainer( path string ) { 
//     c := http.Client{ Timeout: time.Duration(1) * time.Second } 
//     resp, err := c.Get( path ) 
//     if err != nil {
//         fmt.Printf("Error %s", err)
//         return
//     }
//     defer resp.Body.Close()
//     body, err := ioutil.ReadAll(resp.Body) 
// } 

// ----------------------------------------------- 

type Route struct { 
    Name string `json:"name"` 
    Environment map[string]string `json:"env"`
    Image string `json:"image"`
    Provider string `json:"provider"`
    Timeout int `json:"timeout"`
    Retry int `json:"retry"`
    Id string 
} 

// ----------------------------------------------- 

type Container interface { 
    Create ( route Route ) ( state bool, err error ) 
    Check ( route Route ) ( state bool, err error ) 
    Remove ( route Route ) ( state bool, err error ) 
}

// ----------------------------------------------- 

type Containers struct { 
    mutex sync.Mutex 
} 

func ( container *Containers ) Run ( route *Route ) ( err error ) {
    container.mutex.Lock() 
    defer container.mutex.Unlock() 
    if route.Id == "" { 
        _, err := container.Create( route ) 
        if err != nil { 
            return err 
        } 
    } 
    state, err := container.Check( route ) 
    if err != nil { 
        return err 
    } 
    if state == "running" { 
        return nil 
    }
    started, err := container.Start( route ) 
    if err != nil || started == false { 
        return err 
    } 
    for i := 0; i < route.Retry; i++ { 
        time.Sleep( time.Duration( route.Timeout ) * time.Millisecond ) 
        state, err = container.Check( route ) 
        if err != nil { 
            return err 
        } 
        if state == "running" {
            return nil 
        } 
    } 
    return errors.New( "Container has failed to start in time" ) 
} 

func ( container *Containers ) Create ( route *Route ) ( state string, err error ) {
    if route.Image == "" {
        return "undetermined", errors.New( "Image container has null value" ) 
    }
    if route.Name == "" {
        return "undetermined", errors.New( "Name container has null value" ) 
    }
    cmd := exec.Command(
        "docker", 
        "container", 
        "create", 
        "--label", 
        "faasS=true", 
        "--hostname", 
        route.Name, 
        "--env", 
        "FAASS=1", // Environment map[string]string `json:"env"`
        route.Image, 
    )
    o, err := cmd.CombinedOutput() 
    cId := strings.TrimSuffix( string( o ), "\n" ) 
    if err != nil { 
        log.Fatal( "container check ", err ) 
        // return "undetermined", errors.New( cId ) 
    } 
    route.Id = cId 
    return cId, nil 
}

func ( container *Containers ) Check ( route *Route ) ( state string, err error ) { 
    // docker container ls -a --filter 'status=created' --format "{{.ID}}" | xargs docker rm 
    if route.Id == "" {
        return "undetermined", errors.New( "ID container has null string" ) 
    } 
    cmd := exec.Command( 
        "docker", 
        "container", 
        "inspect", 
        "-f", 
        "{{.State.Status}}", 
        route.Id, 
    ) 
    o, err := cmd.CombinedOutput() 
    cState := strings.TrimSuffix( string( o ), "\n" )  
    if err != nil { 
        log.Fatal( "container check ", err ) 
        // return "undetermined", errors.New( cState ) 
    } 
    return cState, nil 
}

func ( container *Containers ) Start ( route *Route ) ( state bool, err error ) {
    if route.Id == "" {
        return false, errors.New( "ID container has null string" ) 
    } 
    cmd := exec.Command( 
        "docker", 
        "container", 
        "restart", 
        route.Id, 
    ) 
    o, err := cmd.CombinedOutput() 
    cId := strings.TrimSuffix( string( o ), "\n" )  
    if err != nil || cId != route.Id { 
        return false, errors.New( cId ) 
    } 
    return true, nil 
}

func ( container *Containers ) Stop ( route *Route ) ( state bool, err error ) { 
    if route.Id == "" {
        return false, errors.New( "ID container has null string" ) 
    } 
    cmd := exec.Command( 
        "docker", 
        "container", 
        "stop", 
        route.Id, 
    ) 
    o, err := cmd.CombinedOutput() 
    cId := strings.TrimSuffix( string( o ), "\n" )  
    if err != nil || cId != route.Id { 
        return false, errors.New( cId ) 
    } 
    return true, nil 
}

func ( container *Containers ) Remove ( route *Route ) ( state bool, err error ) { 
     if route.Id == "" {
        return false, errors.New( "ID container has null string" ) 
    } 
    cmd := exec.Command( 
        "docker", 
        "container", 
        "rm", 
        route.Id, 
    ) 
    o, err := cmd.CombinedOutput() 
    cId := strings.TrimSuffix( string( o ), "\n" )  
    if err != nil || cId != route.Id { 
        return false, errors.New( cId ) 
    } 
    route.Id = "" 
    return true, nil 
}

// ----------------------------------------------- 

func lambdaHandler(w http.ResponseWriter, r *http.Request) { 
    url := r.URL.Path[8:] // "/lambda/" = 8 signes 
    route, err := GLOBAL_CONF.GetRoute( url ) 
    if err != nil { 
        InfoLogger.Println( "unknow desired url :", url, "(", err, ")" ) 
        w.WriteHeader( 404 ) 
        return 
    } 
    InfoLogger.Println( "known desired url :", url ) 
    err = GLOBAL_CONF.Containers.Run( route )
    if err != nil { 
        WarningLogger.Println( "unknow state of container for route :", url, "(", err, ")" ) 
        w.WriteHeader( 503 ) 
        return 
    } 
    DebugLogger.Println( "running container for desired route :", url, "(cId", route.Id, ")" ) 
    fmt.Fprintf(w, "ici : %q", html.EscapeString(url)) 
} 

// ----------------------------------------------- 

func main() { 

    CreateLogger() 
    StartEnv() 

    muxer := http.NewServeMux() 

    UIPath := GLOBAL_CONF.GetParam( "UI" ) 
    if UIPath != "" {
        log.Println( "UI path found :", UIPath ) 
        muxer.Handle( "/", http.FileServer( http.Dir( UIPath ) ) )
    } 
    
    muxer.HandleFunc( "/lambda/", lambdaHandler ) 

    err := http.ListenAndServeTLS(":9090", "server.crt", "server.key", muxer)
    if err != nil {
        log.Println( "ListenAndServe :", err ) 
        os.Exit( ExitUndefined )
    }

}