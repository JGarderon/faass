package httpresponse

import (
  "io"
  "net/http"
  "encoding/json"
  "fmt"
  // "logger"
)

type Response struct {
  Code int 
  MessageError string
  Payload interface{}
  IOFile io.ReadCloser
}

func ( httpR *Response ) Respond( w http.ResponseWriter ) bool { 
  if httpR.Code < 300 {
    if httpR.Payload != nil {
      HTTPResponse, err := json.Marshal( httpR.Payload ) 
      if err != nil { 
        w.WriteHeader( 500 ) 
        w.Header().Set( "Content-type", "application/problem+json" )
        // Logger.Error( "API export conf (Marshal) :", err ) 
        httpR.Code = 500
        return false
      } 
      w.WriteHeader( httpR.Code ) 
      w.Header().Set( "Content-type", "application/json" ) 
      w.Write( HTTPResponse ) 
    } else if httpR.IOFile != nil { 
      w.WriteHeader( httpR.Code ) 
      io.Copy( w, httpR.IOFile )
    } else {
      w.WriteHeader( httpR.Code ) 
    }
    return true 
  } else { 
    w.WriteHeader( httpR.Code ) 
    w.Header().Set( "Content-type", "application/problem+json" )
    w.Write( 
      []byte( 
        fmt.Sprintf( 
          `{"message":"%v"}`, 
          httpR.MessageError, 
        ),
      ),
    )
    return true
  }
}