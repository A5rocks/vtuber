function newField(type) {
    const fields = document.getElementById(type)
    const addRow = fields.children[fields.children.length - 1]
    const uuid = crypto.randomUUID()

    const fieldRow = document.createElement("tr")
    fieldRow.id = uuid

    const name = document.createElement("td")
    const nameInput = document.createElement("input")
    nameInput.type = "text"
    nameInput.form = "searchinfo"
    name.appendChild(nameInput)
    fieldRow.appendChild(name)

    const value = document.createElement("td")
    const valueInput = document.createElement("input")
    valueInput.type = "text"
    valueInput.form = "searchinfo"
    value.appendChild(valueInput)
    fieldRow.appendChild(value)

    const deleteCell = document.createElement("td")
    const deleteText = document.createElement("button")
    deleteText.textContent = "Delete"
    deleteText.onclick = () => removeField(uuid)
    deleteCell.appendChild(deleteText)
    fieldRow.appendChild(deleteCell)

    fields.insertBefore(fieldRow, addRow)
}

function removeField(uuid) {
    const field = document.getElementById(uuid)

    field.parentNode.removeChild(field)
}

function search(event) {
    event.preventDefault()

    const search = {filters: {}, searches: {}};
    
    const filters = document.getElementById("filters")
    const searches = document.getElementById("searches")

    filters.childNodes.forEach((e, i) => {
        if (e.tagName == "TR" && i < filters.childNodes.length - 2) {
            let value = e.childNodes[1].childNodes[0].value
            if (!isNaN(parseInt(value))) value = parseInt(value)
            search.filters[e.childNodes[0].childNodes[0].value] = value
            
        }
    })
    searches.childNodes.forEach((e, i) => {
        if (e.tagName == "TR" && i < searches.childNodes.length - 2) {
            search.searches[e.childNodes[0].childNodes[0].value] = e.childNodes[1].childNodes[0].value
        }
    })

    fetch("/api/search", {
        method: "POST",
        body: JSON.stringify(search)
    }).then((r) => r.json()).then((j) => {
        const listing = document.getElementById("resultlisting")
        const nodes = [];
        listing.childNodes.forEach((n) => nodes.push(n))
        nodes.forEach((n) => listing.removeChild(n))
        for (const result of j) {
            const cell = document.createElement("tr")
            const text = document.createElement("td")
            text.textContent = JSON.stringify(result)

            cell.appendChild(text)
            listing.appendChild(cell)
        }
    }).catch(console.error)
}
