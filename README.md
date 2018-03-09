Hades
===

Hades is a HATEOAS-based HTTP/2 reverse proxy for JSON API servers

## Why?
The JSON API specification makes ample use of `links` objects in its
specification. These links objects contain URLs which enable a client to easily
and automatically traverse documents to get subsequent pages, relationship data
and easily fetch related resources.

With the ever increasing deployment of HTTP/2, these HATEOAS links become much
more relevant, especially when they can be pushed to the client _before the
client even requests them_.

Hades is a reverse proxy that can be deployed in front of any JSON API server.

Clients that are HTTP/2 capable can then send an `X-Push-Please` request header.
The value of the header should `;` delimited [JSONPath selectors](http://goessner.net/articles/JsonPath/index.html#e2) which target
links in the response document. Hades will identify these links and proactively
push these links to the client.
