package configuration

import(
  "errors"
  "os"
  "log"
  "io/ioutil"
  "encoding/json"
  // -----------
  "itinerary"
  "executors"
)

type Conf struct {
  Containers executors.Containers
  Domain string `json:"domain"`
  Authorization string `json:"authorization"`
  IncomingPort int `json:"listen"`
  DelayCleaningContainers int `json:"delay"`
  UI string `json:"ui"`
  TmpDir string `json:"tmp"`
  Prefix string `json:"prefix"`
  Routes map[string]*itinerary.Route `json:"routes"`
}

func (c *Conf) ConfImport( pathRoot string ) bool {
  jsonFileInput, err := os.Open( pathRoot )
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
  json.Unmarshal( byteValue, c )
  if c.DelayCleaningContainers < 5 {
    c.DelayCleaningContainers = 5
    // Logger.Warning( "new value for delay cleaning containers : min 5 (seconds)" ) 
  }
  if c.DelayCleaningContainers > 60 {
    c.DelayCleaningContainers = 60 
    // Logger.Warning( "new value for delay cleaning containers : max 60 (seconds)" ) 
  }
  return true
}

func (c *Conf) GetRoute( key string ) ( route *itinerary.Route, err error ) {
  if route, ok := c.Routes[key]; ok {
    return route, nil
  }
  return nil, errors.New( "unknow itinerary.Routes" )
}

func (c *Conf) Export( pathRoot string ) bool {
  v, err := json.Marshal( c )
  if err != nil {
    log.Fatal( "export conf (Marshal) :", err )
    return false
  }
  jsonFileOutput, err := os.Create( pathRoot )
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