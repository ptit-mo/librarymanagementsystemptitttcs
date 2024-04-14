/**
 * 
 * @param {thumbnailKey} String - Key in data that is used as thumbnail
 * @param {primaryKey} String - Key in data that is used as primary key for query details
 * @param {data} Object - JSON object
 * @param {dataOverlayFormatter} function - Function to create overlay over image. Accept data as parameter
 */
function createGridFromData({
    thumbnailKey,
    data,
    dataOverlayFormatter,
    gridContainerClass = "grid-container",
    gridItemClass = "grid-item",
}) {
    const grid = document.createElement("div");
    grid.classList.add(gridContainerClass);
    data.forEach(item => {
        const gridItem = document.createElement("div");
        gridItem.classList.add(gridItemClass);

        const readonlyDiv = document.createElement("div");
        readonlyDiv.appendChild(createAnImage(item[thumbnailKey]));

        const overlayDiv = document.createElement("div");
        overlayDiv.classList.add("overlay");
        overlayDiv.innerHTML = dataOverlayFormatter(item);
        readonlyDiv.appendChild(overlayDiv);

        const editDiv = document.createElement("div");
        editDiv.classList.add("edit");
        // editDiv.appendChild(createButtonGetDetails(editDiv, { getURL: getDetailURL, id: item.id, next: getNextHandler }))
        // editDiv.appendChild(createButtonDelete({ deleteURL: deleteURL, id: item.id }))

        gridItem.appendChild(readonlyDiv);
        gridItem.appendChild(editDiv);
        grid.appendChild(gridItem);
    });
    return grid.outerHTML
}

function createAnImage(src) {
    const img = document.createElement("img");
    img.src = src;
    img.style.maxWidth = "200px";
    img.style.maxheight = "200px";
    img.style.width = "auto";
    img.style.height = "auto";
    img.style.display = "block";
    return img
}

/**
 * 
 * @param {data} Object - JSON object 
 * @param {tableHeaders} Array - headers for table
 * @returns 
 */
function createTableFromData({ data, tableHeaders }) {
    const table = document.createElement("table");
    let header = `<thead><tr>${tableHeaders.map(header => `<th>${header}</th>`)}</tr></thead>`
    let body = `<tbody>${data.map(row => `<tr>${tableHeaders.map(header => `<td>${row[header]}</td>`)}</tr>`)}</tbody>`
    table.innerHTML = header + body;
    return table
}


function createButtonGetDetails({ getURL, id, next }) {
    const button = document.createElement("button");
    button.innerHTML = "Get Details";
    button.setAttribute('id', `get-details-${id}`);
    button.addEventListener("click", function () {
        alert("get details")
        fetchData(getURL + `?id=${id}`, {
            fetchOptions: {
                method: "GET",
                headers: {
                    "Content-Type": "application/json",
                },
                credentials: "include",
            },
            next: next
        })
    }, false)
    return button
}

function createButtonDelete({ deleteURL, id }) {
    const button = document.createElement("button");
    button.innerHTML = "Delete";
    button.addEventListener("click", function () {
        fetchData(deleteURL + `?id=${id}`, {
            fetchOptions: {
                method: "DELETE",
                headers: {
                    "Content-Type": "application/json",
                },
                credentials: "include",
            },
        })
    })
    return button
}
