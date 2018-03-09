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
example.

A client would like to list the 10 most recent issues and embed user
avatars for every participant in the issue.

In JSON API terms, the data model is structured such that:

1. Issues have a relationship to comments
1. Comments have an author relationship to users
1. User resources have an attribute that specifies a URL for the user's avatar

`issue -> comment -> user -> avatar`

Traditionally, one might use the `include` query parameter to receive a compound
document response. The compound document would embed the chain of resources in a
sinlge response. However, in this scenario, issues do not support includes on
comments. They would be far too numerous&mdash;if 10 issues each have 100 comments,
the compound document would have 1000 resources!

This means that a client will first need to:

1. Fetch the first 10 issues
1. Fetch from the `related` or `relationship` routes for _each_ issue
1. Fetch each the user resource for every unique user
1. Finally, insert or download the avatar image using the user avatar URL

This chain of requests is often called the "waterfall." Each step needs to be
completed before the next step can proceed because the client can't know which
resources to fetch in advance.

In other words, the client can't know which users to fetch if it doesn't know which
comments are on an issue... and the client can't know which comments to fetch without
first fetching the issues.

Hades solves this problem. By specifying the following request, a client can inform
the Hades proxy which resources _it's going to fetch_ and Hades can then proactively
push those resources to the client.

```
GET /api/issues?sort=-created&page[limit]=10

X-Push-Please: $.data[*].relationships.comments.links.related
X-Push-Please: $.data[*].relationships.author.links.related
X-Push-Please: $.data.attributes.avatar.url
```
<sup>Multiple headers are permitted by HTTP/2. Alternatively, values may be
concatenated with `;`</sup>

Hades can use this information to identify the links in the response document
that the client will eventually require. Future versions will permit response
IDs in the `X-Push-Please` header so that link paths can target only specific
responses.

Apart from the client-sent header, _client applications need not be adapted in
any way_. When the client recieves the initial response document, it should still
request the subsequent documents just as it would under HTTP/1.1.

However, because those request response will have already been pushed, they will
already be in a local cache or on the way! That means all responses will appear to
have been reveived as if the client sent all the requests at the same time.

Hades eliminates the waterfall.

🔥
