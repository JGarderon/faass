package api

import (
  "net/http"
  // -----------
  "configuration"
)

func VerifyAuthorization( c *configuration.Conf, r *http.Request ) bool {
  if c.Authorization == "" { 
    if r.Header.Get( "Authorization" ) != "" {
      return false 
    }
    return true
  } else { 
    if r.Header.Get( "Authorization" ) == c.Authorization {
      return true 
    }
    return false 
  }
}