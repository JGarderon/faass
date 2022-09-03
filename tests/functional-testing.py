#!/usr/bin/env python3

import requests
import asyncio
import argparse
import logging
import types

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

  def __init__( self, runner ):
    self.runner = runner
    self.groups = 0 

  async def execute( self ): 
    try: 
      _, pendings = await asyncio.wait(
        [ 
          asyncio.create_task( 
            self.runner.run(), 
            name="runner" 
          ),
          *[ 
            asyncio.create_task( 
              getattr( self, t )(),
              name=t
            ) 
              for t in dir( self )
              if t.startswith( "test_" ) 
          ]
        ], 
        return_when=asyncio.FIRST_COMPLETED
      ) 
      for pending in pendings: 
        if pending.get_name() == "runner": 
          await self.runner.purge( kill=False ) 
    except Exception as err: 
      logger_bar.warningf( f"exception during tests : {err}" ) 
    finally:
      await self.runner.purge( kill=True )

# -----------------------------------------------

class TestsAll( Tests ): 
  url = "https://127.0.0.1:9090/"

  async def test_home_get( self ): 
    await asyncio.sleep( 2 )
    r = requests.get( self.url, verify=False ) #, auth=('user', 'pass'))
    print( r.status_code )
    await asyncio.sleep(0)

# -----------------------------------------------

logger_general.special( """

  FaasS = Function as a (Simple) Service
  ---
  Created by Julien Garderon (Nothus)
  from August 01 to 28, 2022
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
args = parser.parse_args()

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
        ] 
      )
      tests = TestsAll( runner )
      asyncio.run( 
        tests.execute(),
        debug=True 
      )
      if runner.proc is None: 
        raise Exception( "process for run has 'None' value" ) 
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
