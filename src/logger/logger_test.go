package logger 

import (
  "testing"
)

func TestInit( t *testing.T ) {
  l := Logger{}
  l.Init()
} 

func TestTest( t *testing.T ) {
  l := Logger{}
  l.Init()
  l.Test( "ok" ) 
} 