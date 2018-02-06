document.addEventListener("DOMContentLoaded", e => {
  const f = (data, path) => {
    logDocument(data)
    return path ? extract(data, path).catch(console.log) : []
  }

  const g = (link, path, fetchOptions) => {
    return fetch(link, fetchOptions)
      .then(response => {
        return new Promise((resolve, reject) => {
          if (response.ok) {
            resolve(response.json())
          } else {
            reject(response)
          }
        })
      })
      .then(data => {
        return f(data, path)
      })
  }

  const get = (link, {path, children}, buildOptions = true) => {
    const links = Array.isArray(link) ? link : [link]
    const fetchOptions = buildOptions ? buildFetchOptions({path, children}) : {}
    return links.reduce((acc, link) => {
      return children && children.length
        ? children.reduce((acc, child) => {
          return acc.concat(
            g(link, path, fetchOptions).then(innerLinks => get(innerLinks, child, false))
          );
        }, [])
        : g(link, path, fetchOptions).then(innerLinks => {
          innerLinks.forEach(innerLink => g(innerLink, null, {}))
        })
    }, [])
  }

  const thenGet = (path, children = false) => {
    return children
      ? {path, children}
      : {path}
  }

  get('/jsonapi', thenGet("links.node--article", [
    thenGet("data.[].relationships.uid.links.related", [
      thenGet("data.links.self")
    ]),
  ]))
})

function extract(data, path) {
  extractors = {
    "links.node--article": data => {
      return [url2Path(data.links["node--article"])]
    },
    "data.[].relationships.uid.links.related": data => {
      return data.data.map(item => {
        return url2Path(item.relationships.uid.links.related)
      })
    }
  }
  return path && extractors.hasOwnProperty(path)
    ? Promise.resolve(extractors[path](data))
    : Promise.reject(`The path "${path}" could not be found.`)
}

function buildFetchOptions(tree) {
  const buildHeaders = (paths, {path, children}) => {
    return (children && children.length !== 0)
      ? children.reduce(buildHeaders, paths.concat(path))
      : paths.concat(path);
  }
  return {
    headers: {
      "x-push-request": buildHeaders([], tree).join('; ')
    },
  }
}

function url2Path(link) {
  return (new URL(link)).pathname
}

function logDocument(doc) {
  if (Array.isArray(doc.data)) {
    doc.data.forEach(logResource)
  }
  else {
    logResource(doc.data)
  }
}

function logResource(resource) {
  const td = text => {
    const td = document.createElement("td")
    td.appendChild(document.createTextNode(text))
    return td
  }
  const tr = document.createElement("tr")
  tr.appendChild(td(resource.type))
  tr.appendChild(td(resource.id))
  document.getElementById("resource-log").appendChild(tr)
}
