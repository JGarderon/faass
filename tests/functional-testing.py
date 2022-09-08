#!/usr/bin/env python3

import requests
import asyncio
import argparse
import logging
import types
import concurrent.futures
from functools import partial

# -----------------------------------------------

if __name__ != "__main__":
  print( "you can run this script as module" ) 
  exit( 1 ) 

# -----------------------------------------------

class CustomFormatter( logging.Formatter ):
  '''
    source : https://stackoverflow.com/questions/384076/how-can-i-color-python-logging-output
  '''
  special = "\x1b[30;106m"
  grey = "\x1b[38;20m"
  green = "\x1b[32;20m"
  yellow = "\x1b[33;20m"
  red = "\x1b[31;20m"
  bold_red = "\x1b[31;1m"
  reset = "\x1b[0m"
  format_normal = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
  format_all = "%(asctime)s - %(name)s - %(levelname)s - %(message)s (%(filename)s:%(lineno)d)"

  FORMATS = {
      logging.DEBUG: grey + format_all + reset,
      logging.INFO: grey + format_normal + reset,
      (logging.INFO+5): green + format_normal + reset,
      (logging.INFO+7): special + format_normal + reset,
      logging.WARNING: yellow + format_all + reset,
      logging.ERROR: red + format_all + reset,
      logging.CRITICAL: bold_red + format_all + reset
  }

  def format(self, record):
      log_fmt = self.FORMATS.get(record.levelno)
      formatter = logging.Formatter(log_fmt)
      return formatter.format(record)

# -----------------------------------------------

logging.SPECIAL = logging.INFO+7
logging._levelToName[27] = 'SPECIAL'
logging._nameToLevel['SPECIAL'] = 27
logging.OK = logging.INFO+5
logging._levelToName[25] = 'OK'
logging._nameToLevel['OK'] = 25

logger_general = logging.getLogger( "general" )
logger_general.setLevel( logging.DEBUG )
logger_bar = logging.getLogger( "build_and_run" )
logger_bar.setLevel( logging.DEBUG )
ch = logging.StreamHandler()
ch.setLevel( logging.DEBUG )
ch.setFormatter(CustomFormatter())
logger_general.addHandler( ch )
logger_bar.addHandler( ch )

def logging_level_special( self, msg, *args, **kwargs ):
  if self.isEnabledFor( logging.SPECIAL ):
    self._log( logging.SPECIAL, msg, args, **kwargs )

def logging_level_ok( self, msg, *args, **kwargs ):
  if self.isEnabledFor( logging.INFO ):
    self._log( logging.OK, msg, args, **kwargs )

logger_general.special = types.MethodType( logging_level_special, logger_general )
logger_bar.special = types.MethodType( logging_level_special, logger_bar )

logger_general.ok = types.MethodType( logging_level_ok, logger_general )
logger_bar.ok = types.MethodType( logging_level_ok, logger_bar )


# -----------------------------------------------

class TestFail( Exception ): 
  pass 

# -----------------------------------------------

class Runner():

  timeout_terminate = 5

  def __init__( self, cmd, **kwargs ):
    self.cmd = cmd 
    self.env = kwargs
    self.proc = None 
    self.stop = False

  async def __run_read__( self, stream ):
    while stream.at_eof() is not True: 
      try: 
        l = await asyncio.wait_for( 
          stream.readline(), 
          timeout=1 
        )
        l = l.decode( "utf8" ).strip() 
        if l != '': 
          logger_bar.debug( f'"{l}"' )
      except asyncio.TimeoutError:
        pass

  async def __run_wait__( self, timeout=None ):
    if timeout is not None: 
      try:
        await asyncio.wait_for( 
          self.proc.wait(), 
          timeout = timeout
        )
      except asyncio.TimeoutError:
        self.proc.terminate()
        try: 
          await asyncio.wait_for( 
            self.proc.wait(),
            timeout = self.timeout_terminate
          )
        except asyncio.TimeoutError:
          self.proc.kill()
    await self.proc.wait()

  async def run( self, timeout=None ):
    self.proc = await asyncio.create_subprocess_exec(
      *self.cmd,
      stdout=asyncio.subprocess.PIPE,
      stderr=asyncio.subprocess.PIPE, 
      env=self.env
    )
    await asyncio.gather(
      self.__run_read__( self.proc.stdout ),
      self.__run_read__( self.proc.stderr ),
      self.__run_wait__( timeout )
    ) 

  async def purge( self, kill=False ):
    if self.proc is None: 
      return 
    if self.proc.returncode is None: 
      if kill:
        self.proc.kill()
      else: 
        self.proc.terminate()
      await self.proc.wait()

# -----------------------------------------------

class Tests: 

  executor = concurrent.futures.ThreadPoolExecutor( max_workers=15 )

  def __init__( self, runner ):
    self.runner = runner
    self.groups = 0 

  async def __execute__task__( self, fn, t ): 
    try: 
      logger_bar.debug( f"execute test '{t}' (start)" ) 
      await fn() 
      logger_bar.ok( f"execute test '{t}' (pass)" ) 
    except TestFail as err: 
      logger_bar.warning( f"test '{t}' in fail : {err}" ) 
    except Exception as err: 
      logger_bar.critical( f"test '{t}' in unexpected error : {err}" ) 

  async def __wrap_task_fn__( self, fn, *args, **kwargs ): 
    return await self.current_loop.run_in_executor(
      self.executor,
      partial( 
        fn, 
        *args, 
        **kwargs
      ) 
    ) 

  async def __execute__tasks__( self ): 
    await asyncio.sleep( 3 ) # sleep during start of subprocess Faass
    asyncio.gather( 
      *[ 
        self.__execute__task__( 
          getattr( self, t ),
          t
        ) 
          for t in dir( self )
          if t.startswith( "test_" ) 
      ]
    )

  async def execute( self ): 
    try: 
      self.current_loop = asyncio.get_running_loop()
      _, pendings = await asyncio.wait(
        [ 
          asyncio.create_task( 
            self.runner.run(), 
            name="runner" 
          ),
          asyncio.create_task( 
            self.__execute__tasks__(), 
            name="tests"
          ) 
        ], 
        return_when=asyncio.FIRST_COMPLETED
      ) 
      for pending in pendings: 
        if pending.get_name() == "runner": 
          await self.runner.purge( kill=False ) 
        else:
          logger_bar.critical( "not all tests have been played" )
    except Exception as err: 
      logger_bar.warning( f"exception during tests : {err}" ) 
    finally:
      await self.runner.purge( kill=True )
      if self.runner.proc.returncode != 0: 
        logger_bar.critical( f"runner of faass stopped with '{self.runner.proc.returncode}' code" ) 

# -----------------------------------------------

class TestsAll( Tests ): 
  url = "https://127.0.0.1:9090"
  path_lambda = "/lambda"
  path_api = "/api"
  auth = ( 'admin', 'azerty' )

  async def test_home_get( self ): 
    r = await self.__wrap_task_fn__(
      requests.get,
      self.url, 
      verify=False 
    ) 
    if r.status_code != 200: 
      raise TestFail( f"test home in error, HTTP status : {r.status_code} (expected : 200)" )

  async def test_lambda_functions_get( self ): 
    m = "echo"
    r = await self.__wrap_task_fn__(
      requests.get,
      f"{self.url}{self.path_lambda}/example-function", 
      data=m, 
      verify=False
    ) 
    if r.status_code != 200: 
      raise TestFail( f"test lambda-functions in error, HTTP status : {r.status_code} (expected : 200)" )
    if r.text != m: 
      raise TestFail( f"test lambda-functions in error, no echo found (expected : '{m}')" )

  async def test_lambda_services_get( self ): 
    r = await self.__wrap_task_fn__(
      requests.get,
      f"{self.url}{self.path_lambda}/example-service", 
      verify=False, 
      auth=self.auth 
    )
    if r.status_code != 200: 
      raise TestFail( f"test lambda-service in error, HTTP status : {r.status_code} (expected : 200)" )

  async def test_lambda_url_invalid_get( self ): 
    r = await self.__wrap_task_fn__(
      requests.get,
      f"{self.url}{self.path_lambda}/?", 
      verify=False 
    )
    if r.status_code != 400: 
      raise TestFail( f"test lambda-? (not found) in error, HTTP status : {r.status_code} (expected : 400)" )

  async def test_lambda_url_notfound_get( self ): 
    r = await self.__wrap_task_fn__(
      requests.get,
      f"{self.url}{self.path_lambda}/notfound", 
      verify=False 
    ) 
    if r.status_code != 404: 
      raise TestFail( f"test lambda-? (not found) in error, HTTP status : {r.status_code} (expected : 404)" )

# -----------------------------------------------

logger_general.special( """

  FaasS = Function as a (Simple) Service
  ---
  Created by Julien Garderon (Nothus)
  from August 01 to September 06, 2022
  MIT Licence
  ---
  This is a POC - Proof of Concept -, based on the idea of the OpenFaas project
  /!\\ NOT INTENDED FOR PRODUCTION, only dev /!\\
""" ) 

try: 
  logger_general.ok( "all is ready for work !" ) 
except Exception: 
  logger_general.critical( "faild to log with 'ok' method" )
  exit( 1 )

# -----------------------------------------------

parser = argparse.ArgumentParser(
  description="builds and functionally tests the FAASS' program"
)
parser.add_argument( 
  "--build", 
  action='store_true',
  help="build the program, stop this script if failed"
)
parser.add_argument( 
  "--run", 
  action='store_true',
  help="run the program"
)
parser.add_argument( 
  "--origin-path", 
  type=str, 
  default=".",
  help="set the path of main.go"
)
parser.add_argument( 
  "--cache-path", 
  type=str, 
  default=".",
  help="set the path of cache"
)
parser.add_argument( 
  "--conf-path", 
  type=str, 
  default="./conf.json",
  help="set the path of cache"
)
parser.add_argument( 
  "--output-file", 
  type=str, 
  default="./faass",
  help="set the name of output builder"
)
parser.add_argument( 
  "--log", 
  type=str, 
  default="INFO",
  help="set the log level (DEBUG, INFO, ...)"
)
args = parser.parse_args()

log_level = args.log.upper() 
numeric_level = getattr( logging, log_level, None )
if not isinstance( numeric_level, int ):
    raise logger_general.critical( f"Invalid log level: {log_level}" )
logger_general.setLevel( level=numeric_level )
logger_bar.setLevel( level=numeric_level )

try: 
  if args.build:
    logger_general.info( "build asked" ) 
    builder_env = {
      "GOPATH": args.origin_path,
      "GOCACHE": args.cache_path,
      "GO111MODULE": "off"
    }
    [ logger_general.debug( f"build env {key}={builder_env[key]}" ) for key in builder_env ]
    builder = Runner( 
      [ 
        "go", "build", 
          "-v", 
          "-o", args.output_file
      ], 
      **builder_env
    ) 
    builder.timeout_terminate = 60
    asyncio.run( builder.run() ) 
    if builder.proc.returncode != 0:
      logger_general.critical( "failed for build programm" ) 
      exit( 1 )
    else: 
      logger_general.ok( "build finished" ) 
  if args.run:
    try:
      runner = Runner( 
        [ 
          args.output_file, 
            "--conf", args.conf_path 
        ],
        PATH="/usr/bin/docker" 
      )
      tests = TestsAll( runner )
      asyncio.run( 
        tests.execute(),
        debug=False 
      )
      if runner.proc is None: 
        raise Exception( "process for run has 'None' value at end of script" ) 
      elif runner.proc.returncode == 0: 
        logger_bar.ok( "run without error" ) 
      else: 
        logger_bar.ok( f"run without critical error but return code is {runner.proc.returncode}" ) 
    except Exception as err: 
      logger_bar.critical( f"unexpected error during run : {err}" )
except KeyboardInterrupt:
  pass 

# print(f'--> {runner.cmd!r} ({runner.proc.pid}) exited with {runner.proc.returncode}')
# -----------------------------------------------

logger_general.special( "ending of script" ) 
