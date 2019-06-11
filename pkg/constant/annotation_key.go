package constant

var AnnotationKeys map[int]string

const (
	API                                       int32 = 12
	API_METADATA                                    = 13
	RETURN_DATA                                     = 14
	API_TAG                                         = 10015
	ERROR_API_METADATA_ERROR                        = 10000010
	ERROR_API_METADATA_AGENT_INFO_NOT_FOUND         = 10000011
	ERROR_API_METADATA_IDENTIFIER_CHECK_ERROR       = 10000012
	ERROR_API_METADATA_NOT_FOUND                    = 10000013
	ERROR_API_METADATA_DID_COLLSION                 = 10000014
	SQL_ID                                          = 20
	SQL                                             = 21
	SQL_METADATA                                    = 22
	SQL_PARAM                                       = 23
	SQL_BINDVALUE                                   = 24
	STRING_ID                                       = 30
	HTTP_URL                                        = 40
	HTTP_PARAM                                      = 41
	HTTP_PARAM_ENTITY                               = 42
	HTTP_COOKIE                                     = 45
	HTTP_STATUS_CODE                                = 46
	HTTP_INTERNAL_DISPLAY                           = 48
	HTTP_IO                                         = 49
	MESSAGE_QUEUE_URI                               = 100
	ARGS0                                           = -1
	ARGS1                                           = -2
	ARGS2                                           = -3
	ARGS3                                           = -4
	ARGS4                                           = -5
	ARGS5                                           = -6
	ARGS6                                           = -7
	ARGS7                                           = -8
	ARGS8                                           = -9
	ARGS9                                           = -10
	ARGSN                                           = -11
	CACHE_ARGS0                                     = -30
	CACHE_ARGS1                                     = -31
	CACHE_ARGS2                                     = -32
	CACHE_ARGS3                                     = -33
	CACHE_ARGS4                                     = -34
	CACHE_ARGS5                                     = -35
	CACHE_ARGS6                                     = -36
	CACHE_ARGS7                                     = -37
	CACHE_ARGS8                                     = -38
	CACHE_ARGS9                                     = -39
	CACHE_ARGSN                                     = -40
	EXCEPTION                                       = -50
	EXCEPTION_CLASS                                 = -51
	UNKNOWN                                         = -9999
	ASYNC                                           = -100
	PROXY_HTTP_HEADER                               = 300
	REDIS_IO                                        = 310
	JSON_LIB_ANNOTATION_KEY_JSON_LENGTH             = 9002
	GSON_ANNOTATION_KEY_JSON_LENGTH                 = 9000
	JACKSON_ANNOTATION_KEY_LENGTH_VALUE             = 9001
	// Dubbo
	DUBBO_ARGS                  = 90
	DUBBO_RESULT                = 91
	DUBBO_RPC                   = 92
	DUBBO_STATUS_ANNOTATION_KEY = 93
	DUBBO_CONSUMER_HOST_KEY     = 94
)

func initAnnotationKeys() {
	AnnotationKeys = make(map[int]string)
	AnnotationKeys[12] = "API"
	AnnotationKeys[13] = "API-METADATA"
	AnnotationKeys[14] = "RETURN_DATA"
	AnnotationKeys[10015] = "API-TAG"
	AnnotationKeys[10000010] = "API-METADATA-ERROR"
	AnnotationKeys[10000011] = "API-METADATA-AGENT-INFO-NOT-FOUND"
	AnnotationKeys[10000012] = "API-METADATA-IDENTIFIER-CHECK_ERROR"
	AnnotationKeys[10000013] = "API-METADATA-NOT-FOUND"
	AnnotationKeys[10000014] = "API-METADATA-DID-COLLSION"
	AnnotationKeys[20] = "SQL-ID"
	AnnotationKeys[21] = "SQL"
	AnnotationKeys[22] = "SQL-METADATA"
	AnnotationKeys[23] = "SQL-PARAM"
	AnnotationKeys[24] = "SQL-BindValue"
	AnnotationKeys[30] = "STRING_ID"
	AnnotationKeys[40] = "http.url"
	AnnotationKeys[41] = "http.param"
	AnnotationKeys[42] = "http.entity"
	AnnotationKeys[45] = "http.cookie"
	AnnotationKeys[46] = "http.status.code"
	AnnotationKeys[48] = "http.internal.display"
	AnnotationKeys[49] = "http.io"
	AnnotationKeys[100] = "message.queue.url"
	AnnotationKeys[-1] = "args[0]"
	AnnotationKeys[-2] = "args[1]"
	AnnotationKeys[-3] = "args[2]"
	AnnotationKeys[-4] = "args[3]"
	AnnotationKeys[-5] = "args[4]"
	AnnotationKeys[-6] = "args[5]"
	AnnotationKeys[-7] = "args[6]"
	AnnotationKeys[-8] = "args[7]"
	AnnotationKeys[-9] = "args[8]"
	AnnotationKeys[-10] = "args[9]"
	AnnotationKeys[-11] = "args[N]"

	AnnotationKeys[-30] = "cached_args[0]"
	AnnotationKeys[-31] = "cached_args[1]"
	AnnotationKeys[-32] = "cached_args[2]"
	AnnotationKeys[-33] = "cached_args[3]"
	AnnotationKeys[-34] = "cached_args[4]"
	AnnotationKeys[-35] = "cached_args[5]"
	AnnotationKeys[-36] = "cached_args[6]"
	AnnotationKeys[-37] = "cached_args[7]"
	AnnotationKeys[-38] = "cached_args[8]"
	AnnotationKeys[-39] = "cached_args[9]"
	AnnotationKeys[-40] = "cached_args[N]"

	AnnotationKeys[-50] = "Exception"
	AnnotationKeys[-51] = "ExceptionClass"
	AnnotationKeys[-9999] = "UNKNOWN"
	AnnotationKeys[-100] = "Asynchronous Invocation"
	AnnotationKeys[300] = "PROXY_HTTP_HEADER"

	AnnotationKeys[310] = "REDIS_IO"

	AnnotationKeys[9002] = "json-lib.json.length"
	AnnotationKeys[9000] = "gson.json.length"
	AnnotationKeys[9001] = "jackson.json.length"

	//Dubbo
	AnnotationKeys[90] = "dubbo.args"
	AnnotationKeys[91] = "dubbo.result"
	AnnotationKeys[92] = "dubbo.rpc"
	AnnotationKeys[93] = "dubbo.result.status"
	AnnotationKeys[94] = "dubbo.consumer.host"
}
