package shell

import (
  "context"
  "os/exec"
  "errors"
  "fmt"
  // -----------
)

func ExecuteRequest ( ctx context.Context, routeName string, scriptPath string, scriptCmd []string, routeEnv map[string]string ) ( cmd *exec.Cmd, err error ) {
  if routeName == "" {
    return nil, errors.New( "route's name undefined" ) 
  } 
  if scriptPath == "" {
    return nil, errors.New( "script's path undefined" ) 
  } 
  cmd = exec.CommandContext( 
    ctx, 
    scriptPath, 
    scriptCmd... 
  )
  envLocal := []string{
    fmt.Sprintf( "FAAS_ROUTE=%v", routeName ),
  }
  for envName, envValue := range routeEnv {
    envLocal = append( 
      envLocal, 
      fmt.Sprintf( "%v=%v", envName, envValue ),
    ) 
  }
  cmd.Env = envLocal
  return cmd, nil 
}
