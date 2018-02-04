document.addEventListener("DOMContentLoaded", e => {
  const headers = {"headers": {"x-push-request": "data.[].relationships.uid.links.self"}}
  const logResource = resource => {
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
  fetch('/jsonapi/node/article', headers)
    .then(data => data.json())
    .then(json => json.data.map(node => {
      logResource(node)
      return node.relationships.uid.links.self
    }))
    .then(links => links.map(link => (new URL(link)).pathname))
    .then(paths => Promise.all(paths.map(path => fetch(path)
      .then(data => data.json())
    )))
    .then(users => users.map(user => user.data).map(logResource))
});
