// Package docs Monita REST API.
//
// This is the documentation of the Monita REST API.
//
//	# Authentication
//	Monita uses two primary token types:
//	__clientToken__: a client receives messages and manages account resources.
//	__appToken__: an application sends messages.
//
//	The token can be transmitted in a header named `X-Monita-Key`, in a query parameter named `token`, or
//	through an `Authorization` header with the value prefixed with `Bearer` (for example, `Bearer randomtoken`).
//	For Gotify compatibility, `X-Gotify-Key` remains accepted as a legacy alias.
//	Basic auth is also available for supported login/elevation flows.
//
//	\---
//
//	Found a bug or have some questions? [Create an issue on GitHub](https://github.com/gigabytegrove/monita/issues)
//
//	    Schemes: http, https
//	    Host: localhost
//	    Version: 1.3.7
//	    License: MIT https://github.com/gigabytegrove/monita/blob/master/LICENSE
//
//	    Consumes:
//	    - application/json
//
//	    Produces:
//	    - application/json
//
//	    SecurityDefinitions:
//	       appTokenQuery:
//	          type: apiKey
//	          name: token
//	          in: query
//	       clientTokenQuery:
//	          type: apiKey
//	          name: token
//	          in: query
//	       appTokenHeader:
//	          type: apiKey
//	          name: X-Monita-Key
//	          in: header
//	       clientTokenHeader:
//	          type: apiKey
//	          name: X-Monita-Key
//	          in: header
//	       appTokenAuthorizationHeader:
//	          type: apiKey
//	          name: Authorization
//	          in: header
//	          description: >-
//	              Enter an application token with the `Bearer` prefix, e.g. `Bearer Axxxxxxxxxx`.
//	       clientTokenAuthorizationHeader:
//	          type: apiKey
//	          name: Authorization
//	          in: header
//	          description: >-
//	              Enter a client token with the `Bearer` prefix, e.g. `Bearer Cxxxxxxxxxx`.
//	       basicAuth:
//	          type: basic
//
//	swagger:meta
package docs
