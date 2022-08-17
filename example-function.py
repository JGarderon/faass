import sys, json, os

payload_headers = str( json.dumps( {
	"code": 202, 
	"headers": {
		"test": "ok"
	}
} ) ).encode( 'utf-8' )

sys.stdout.buffer.write( 
	len( payload_headers ).to_bytes( 4, byteorder='big', signed=False )
) 
sys.stdout.buffer.write( 
	payload_headers 
) 

# payload_body = str( dict( os.environ ) ).encode( 'utf-8' )

# sys.stdout.buffer.write( 
# 	len( payload_body ).to_bytes( 4, byteorder='big', signed=False )
# ) 
# sys.stdout.buffer.write( 
# 	payload_body 
# ) 
