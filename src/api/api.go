package api

import (
  "net/http"
  // -----------
  "configuration"
)

func VerifyAuthorization( c *configuration.Conf, r *http.Request ) bool {
  return ( r.Header.Get( "Authorization" ) == c.Authorization ) 
}