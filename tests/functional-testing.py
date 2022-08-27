#!/usr/bin/env python3

import requests
import asyncio

# -----------------------------------------------

if __name__ != "__main__":
  print( "you can run this script as module" ) 
  exit( 1 ) 

# -----------------------------------------------

class Runner():

  def __init__( self, cmd ):
    self.cmd = cmd 
    self.proc = None 

  async def run_read( self, stream ):
    while True: 
      l = await stream.readline()
      if l == b'': 
        break
      print( ">>> ", l ) 

  async def run_wait( self, time_out=None ):
    if time_out is None: 
      await self.proc.wait()
    else:
      await asyncio.wait_for( 
        self.proc.wait(), 
        timeout=time_out
      )

  async def run( self, time_out=None ):
    self.proc = await asyncio.create_subprocess_shell(
      self.cmd,
      stdout=asyncio.subprocess.PIPE,
      stderr=asyncio.subprocess.PIPE)
    await asyncio.gather(
      self.run_read( self.proc.stdout ),
      self.run_read( self.proc.stderr ),
      self.run_wait( time_out )
    ) 

# -----------------------------------------------

print( "start of tests" ) 

# -----------------------------------------------

try: 
  runner = Runner( './faass' )
  asyncio.run( runner.run( 5 ) )
  print(f'[{runner.cmd!r} exited with {runner.proc.returncode}]')
except KeyboardInterrupt: 
  pass 

# -----------------------------------------------

print( "end of tests" ) 
