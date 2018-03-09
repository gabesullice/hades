Hades
===

Hades is an *experimental* HATEOAS-based HTTP/2 reverse proxy for JSON API backends.

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
The values of this header should be [JSONPath selectors](http://goessner.net/articles/JsonPath/index.html#e2) which target
links in the response document. Hades will identify these links and proactively
push these links to the client.

## Example
Take a JSON API server with a `issue`, `comments` and `user` resource types.
Your client would like to list recent issues and embed user avatars for every
participant in an issue. In JSON API terms, issues have a relationship to
comments and comments have an author relationship to users. User resources also
have include a link to their avatar:

`issues -> comments -> user -> avatar`

Traditionally, one might use the `include` query parameter to receive a compound
document response from the JSON API server. However, in this scenario, issues do
not support includes on comments. They would be far too numerous&mdash;if 10
issues each have 100 comments, the compound document would have 1000 resources.

This means that a client will first need to fetch the issues, then visit the
`related` or `relationship` endpoints for each issue and possibly then also
fetch the user resources. Finally, the avatar URL will need to be added to the
DOM and the image will be requested. This is called the "waterfall."

With Hades as a reverse proxy, a client could specify the following request:

```
GET /api/issues?sort=-created&page[limit]=10

X-Push-Please: $.data[*].relationships.comments.links.related
X-Push-Please: $.data[*].relationships.author.links.related
X-Push-Please: $.data.attributes.avatar.url
```
<sup>Multiple headers are permitted by HTTP/2. The values may also be
concatenated with `;`</sup>

Hades will identify any links in the response document and initiate server
pushes for those resources. Those responses will also be evaluated for links
(future version will permit response IDs so that the link paths don't apply to
all responses).

Apart from the client-sent header, client applications need not adapt in any
way. When the client recieves the initial response document, it should still
request the subsequent documents as before. However, these responses will
already be in a local cache or already on the way!

What this means is the the "waterfall" effect is removed and all responses
should be reveived as if the client sent parallel requests for all resources at
once.
