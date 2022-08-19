package executors

import(
  "context"
  "os"
  "os/exec"
  "errors"
  "time"
  "path/filepath"
  "strings"
  "fmt"
    // -----------
  "itinerary"
  "logger"
)

type Containers struct {
  Logger *logger.Logger
}

func ( container *Containers ) ExecuteRequest ( ctx context.Context, routeName string, scriptPath string, fileEnvPath string, imageContainer string, scriptCmd []string ) ( cmd *exec.Cmd, err error ) {
  if routeName == "" {
    return nil, errors.New( "image's name undefined" ) 
  } 
  if scriptPath == "" {
    return nil, errors.New( "script's path undefined" ) 
  } 
  if fileEnvPath == "" {
    return nil, errors.New( "env file's path undefined" ) 
  } 
  if imageContainer == "" {
    return nil, errors.New( "env file's path undefined" ) 
  } 
  args := []string{ 
    "run", 
      "-i", 
      "--rm", 
      "-a", "stderr", 
      "-a", "stdout", 
      "-a", "stdin", 
      "--label", "faass=true",
      "--mount", "type=bind,source="+scriptPath+",target=/function,readonly",
      "--hostname", routeName,
      "--env-file", fileEnvPath,
      imageContainer, 
  } 
  args = append(args, scriptCmd[:]...)
  cmd = exec.CommandContext( ctx, "docker", args... )
  return cmd, nil 
} 

func ( container *Containers ) Run ( tmpDir string, route *itinerary.Route ) ( err error ) {
  route.Mutex.Lock()
  if route.Id == "" {
    _, err := container.Create( tmpDir, route )
    if err != nil {
      route.Mutex.Unlock()
      return err
    } 
  }
  route.LastRequest = time.Now()
  route.Mutex.Unlock()
  state, err := container.Check( route )
  if err != nil {
    return err
  }
  if state == "running" {
    return nil
  }
  started, err := container.Start( route )
  cIpAdress, err := container.GetInfos(
    route,
    "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}",
  )
  if err != nil {
    return err
  }
  route.IpAdress = cIpAdress
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

func ( container *Containers ) Create ( tmpDir string, route *itinerary.Route ) ( state string, err error ) {
  if route.Image == "" {
    return "failed", errors.New( "Image container has null value" )
  }
  if route.Name == "" {
    return "failed", errors.New( "Name container has null value" )
  }
  fileEnvPath := filepath.Join(
    tmpDir,
    route.Name+".env", 
  )
  fileEnv, err := os.Create( fileEnvPath )
  if err != nil {
    container.Logger.Error( "unable to create container file env : ", err )
    return "failed", errors.New( "env file for container failed" )
  }
  for key, value := range route.Environment {
    fileEnv.WriteString( key+"="+value+"\n" )
  }
  fileEnv.Close()
  pathContainerTmpDir := filepath.Join(
    tmpDir,
    route.Name, 
  )
  if err := os.MkdirAll( pathContainerTmpDir, os.ModePerm ); err != nil {
    container.Logger.Error( "unable to create tmp dir for container : ", err )
    return "failed", errors.New( "tmp dir for container failed" )
  }
  cmd := exec.Command(
    "docker", "container", "create",
      "--label", "faass=true",
      "--mount", "type=bind,source="+pathContainerTmpDir+",target=/hostdir",
      "--hostname", route.Name,
      "--env-file", fileEnvPath,
      route.Image,
  )
  o, err := cmd.CombinedOutput()
  cId := strings.TrimSuffix( string( o ), "\n" )
  if err != nil {
    container.Logger.Error( "container create in error : ", err )
    return "undetermined", errors.New( cId )
  }
  route.Id = cId 
  cIP, err := container.GetInfos( route, "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}" )
  if err != nil {
    container.Logger.Errorf( "container '%v' (cId %v) check failed", route.Name, route.Id, err )
    return "undetermined", errors.New( cIP )
  }
  route.IpAdress = cIP
  return cId, nil
}

func ( container *Containers ) Check ( route *itinerary.Route ) ( state string, err error ) {
  // docker container ls -a --filter 'status=created' --format "{{.ID}}" | xargs docker rm
  if route.Id == "" {
    return "undetermined", errors.New( "ID container has null string" )
  }
  cState, err := container.GetInfos( route, "{{.State.Status}}" )
  if err != nil {
    container.Logger.Errorf( "container '%s' check failed", route.Name, err )
    return "undetermined", errors.New( cState )
  }
  return cState, nil
}

func ( container *Containers ) Start ( route *itinerary.Route ) ( state bool, err error ) {
  if route.Id == "" {
    return false, errors.New( "ID container has null string" )
  }
  cmd := exec.Command(
    "docker", "container", "restart",
      route.Id,
  )
  o, err := cmd.CombinedOutput()
  cId := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil || cId != route.Id {
    return false, errors.New( cId )
  }
  return true, nil
}

func ( container *Containers ) Stop ( route *itinerary.Route ) ( state bool, err error ) {
  if route.Id == "" {
    return false, errors.New( "ID container has null string" )
  }
  cmd := exec.Command(
    "docker", "container", "stop",
      route.Id,
  )
  o, err := cmd.CombinedOutput()
  cId := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil || cId != route.Id {
    return false, errors.New( cId )
  }
  return true, nil
}

func ( container *Containers ) Remove ( route *itinerary.Route ) ( state bool, err error ) {
   if route.Id == "" {
    return false, errors.New( "ID container has null string" )
  }
  cmd := exec.Command(
    "docker", "container", "rm",
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

func ( container *Containers ) GetInfos ( route *itinerary.Route, pattern string ) ( infos string, err error ) {
   if route.Id == "" {
    return "", errors.New( "ID container has null string" )
  }
  cmd := exec.Command(
    "docker", "container", "inspect",
      "-f", pattern,
        route.Id,
  )
  o, err := cmd.CombinedOutput()
  cInfos := strings.TrimSuffix( string( o ), "\n" ) 
  if err != nil {
    return "", errors.New( 
      fmt.Sprintf( 
        "failed to get infos for route %v", 
        route.Id, 
      ), 
    ) 
  }
  return cInfos, nil
}