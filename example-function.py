#!/usr/bin/env python3

import sys, json, os

# print("démarrage", file=sys.stderr) 

payload_headers = str( json.dumps( {
	"code": 202, 
	"headers": {
		"Content-type": "text/plain", 
		"test": "ok"
	}
} ) ).encode( 'utf-8' )

sys.stdout.buffer.write( 
	len( payload_headers ).to_bytes( 4, byteorder='big', signed=False )
) 
sys.stdout.buffer.write( 
	payload_headers 
) 

# payload_body = str( json.dumps( dict( os.environ ) ) ).encode( 'utf-8' )
payload_body = str( sys.stdin.read() ).encode( 'utf-8' )

# print("payload_body", sys.stdin.readlines(), file=sys.stderr) 

sys.stdout.buffer.write( 
	len( payload_body ).to_bytes( 4, byteorder='big', signed=False )
) 
sys.stdout.buffer.write( 
	payload_body 
) 
