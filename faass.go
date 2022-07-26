package main

import (
    "encoding/json" 
    "io/ioutil"
    "os"
    "reflect"
    "errors" 
    "flag"
    "net/http"
    "log"
    // "fmt"
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

// func redirectionHandler(w http.ResponseWriter, r *http.Request) {
//     url := GLOBAL_DOMAIN + r.URL.Path[10:] // "/redirect/" = 10 signes 
//     http.Redirect(w, r, url, 301)
// }

// func lambdaHandler(w http.ResponseWriter, r *http.Request) {
//     url := GLOBAL_DOMAIN + r.URL.Path[10:] // "/lambda/" = 8 signes 
//     http.Redirect(w, r, url, 301)
// }

// ----------------------------------------------- 

type Conf struct {
    Domain string `json:"domain"`
    UI string `json:"ui"`
    Containers string `json:"containers"`
    TmpDir string `json:"dirtmp"`
    Prefix string `json:"prefix"`
    Routes map[string]Route `json:"routes"`
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

func (c *Conf) GetRoute( key string ) (route Route, err error) { 
    if route, ok := c.Routes[key]; ok {
        return route, nil
    } 
    return Route{}, errors.New( "unknow routes" ) 
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

type Route struct { 
    Name string `json:"name"` 
    Environment map[string]string `json:"env"`
    Image string `json:"image"`
    Provider string `json:"provider"`
} 

// ----------------------------------------------- 

func CreateEnv() bool {
    uiTmpDir := "./content" 
    pathTmpDir := "./tmp" 
    GLOBAL_CONF = Conf { 
        Domain: "https://localhost", 
        UI: uiTmpDir, 
        Containers: "podman", 
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


func main() { 

    StartEnv() 

    muxer := http.NewServeMux() 

    UIPath := GLOBAL_CONF.GetParam( "UI" ) 
    if UIPath != "" {
        log.Println( "UI path found :", UIPath ) 
        muxer.Handle( "/", http.FileServer( http.Dir( UIPath ) ) )
    }
    
    // muxer.HandleFunc("/redirect/", redirectionHandler) 
    
    // muxer.HandleFunc("/lambda/", lambdaHandler)     

    err := http.ListenAndServeTLS(":9090", "server.crt", "server.key", muxer)
    if err != nil {
        log.Println( "ListenAndServe :", err ) 
        os.Exit( ExitUndefined )
    }
    
}