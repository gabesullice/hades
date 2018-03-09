Hades
===

Hades is an **experimental** HATEOAS-based HTTP/2 reverse proxy for JSON API backends.

## Why?
The JSON API specification makes ample use of `links` objects in its
specification. These links objects contain URLs which enable a client to easily
and automatically traverse documents to fetch subsequent pages, relationship
data and related resources.

With the ever increasing deployment of HTTP/2, these HATEOAS links become much
more relevant, especially when they can be pushed to the client _before the
client even requests them_.

Hades is an intelligent reverse proxy that can be deployed in front of any
JSON API server.

Clients that are HTTP/2 capable can then send an `X-Push-Please` request header
to your application. The values of this header should be comprised of [JSONPath selectors](http://goessner.net/articles/JsonPath/index.html#e2)
which target links in the expected response document. Hades will identify these
links and proactively push the linked resources to the client.

Clients that are not HTTP/2 capable can also send these headers, Hades will
simply not push the responses. Future versions of Hades might make further
optimizations (like using the request to warm a server-side application cache).

## Example
Take a JSON API server with `issue`, `comment` and `user` resource types as an
example. A client would like to list the 10 most recent issues and embed user
avatars for every participant in the issue. In JSON API terms, issues have a
relationship to comments and comments have an author relationship to users. User
resources also a download URL attribute for each user's avatar.

`issue -> comment -> user -> avatar`

Traditionally, one might use the `include` query parameter to receive a compound
document response from the JSON API server. However, in this scenario, issues do
not support includes on comments. They would be far too numerous&mdash;if 10
issues each have 100 comments, the compound document would have 1000 resources!

This means that a client will first need to fetch the issues, then visit the
`related` or `relationship` endpoints for each issue and then also fetch each
user resource for every unique user. Finally, the avatar URL will need to be
added to the DOM via an `img src` attribute (which will trigger a request for 
the image). This chain of requests is often called the "waterfall."

With Hades as a reverse proxy, a client could specify the following request:

```
GET /api/issues?sort=-created&page[limit]=10

X-Push-Please: $.data[*].relationships.comments.links.related
X-Push-Please: $.data[*].relationships.author.links.related
X-Push-Please: $.data.attributes.avatar.url
```
<sup>Multiple headers are permitted by HTTP/2. Alternatively, values may be
concatenated with `;`</sup>

Using this information, Hades will identify any links in the response document
and initiate server pushes for those resources. Those responses will also be
evaluated for links and those responses will be pushed as well (future version
will permit response IDs in the `X-Push-Please` header so that link paths can
target only specific responses).

Apart from the client-sent header, client applications need not be adapted in
any way. When the client recieves the initial response document, it should still
request the subsequent documents just as it would under HTTP/1.1. However, these
responses will already be in a local cache or already on the way! In fact, all
responses will appear to have been reveived as if the client sent parallel requests
for every resources at once.


What this means is that Hades eliminates the waterfall.

🔥
