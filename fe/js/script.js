async function fetchJSONData(url, fetchOptions) {
    const response = await fetch(url, fetchOptions);
    if (!response.ok) {
        if (response.status === 401) {
            window.location.href = "/fe/login.html"
            throw new Error("Unauthorized")
        }
        if (response.status === 404) {
            throw new Error(`${url} not found`)
        }
        const errorMessage = await response.json();
        throw new Error(errorMessage["message"]);
    }
    return response.json();
}

async function fetchData(url, fetchOptions) {
    try {
        let response = await fetchJSONData(url, fetchOptions)
        if (response == undefined) {
            updateStatusBox(url, { "message": "No data" })
            return []
        }
        if (response["message"] != undefined) {
            updateStatusBox(url, response)
        }
        return response
    } catch (error) {
        updateStatusBox(url, { "message": error.message })
        throw error
    }
}

function updateStatusBox(url, data) {
    const statusBox = document.getElementById("status-box");
    if (statusBox == null) {
        return
    }
    const className = data["message"] == undefined ? "container goodresponse" : "container badresponse"
    let content = data["message"] == undefined ? `${url} succeeded` : data["message"]
    statusBox.className = className
    statusBox.innerHTML = content
    statusBox.style.display = "block";
    console.log(data); // Handle response data
    setTimeout(() => {
        statusBox.style.display = "none";
    }, 2000)
}

function generateNavBar() {
    const navBar = document.getElementById('navbar');
    const notLoggedInYet = (window.localStorage.getItem('user_type') == null)
    const nonAdminUser = (notLoggedInYet || window.localStorage.getItem('user_type') == 'borrower')
    navBar.innerHTML = `
    <div class="container-fluid">
        <nav class="collapse navbar-collapse bs-navbar-collapse">
            <ul class="nav navbar-nav nav-pills">
                <li><a href="/fe/index.html">Home</a></li>
                ${notLoggedInYet ? `<li><a href="/fe/login.html">Login</a></li>` : ''}
                ${nonAdminUser ? '' : `<li><a href="/fe/book.html">Book</a></li>`}
                ${nonAdminUser ? '' : `<li><a href="/fe/user.html">User</a></li>`}
                <li><a href="/fe/borrow_history.html">Borrow History</a></li>
                <li><a href="/fe/logout.html">Logout</a></li>
        </nav>
    </div>
    `;
}

/**
 * Helper function for POSTing data as JSON with fetch.
 *
 * @param {Object} options
 * @param {string} options.url - URL to POST data to
 * @param {FormData} options.formData - `FormData` instance
 * @return {Object} - Response body from URL that was POSTed to
 */
var postFormDataAsJson = async ({
    url,
    formData,
    intInputFields = [],
}) => {
    const plainFormData = Object.fromEntries(formData.entries());
    for (const field of intInputFields) {
        plainFormData[field] = parseInt(plainFormData[field])
    }
    const formDataJsonString = JSON.stringify(plainFormData);

    const fetchOptions = {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
        },
        body: formDataJsonString,
    };

    return await fetchData(url, fetchOptions);
}
/**
 * Event handler for a form submit event.
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement/submit_event
 * @example const exampleForm = document.getElementById("example-form");
 *          exampleForm.addEventListener("submit", handleFormSubmit);
 * @param {SubmitEvent} event
 */
var handleFormSubmit = async (event, next) => {
    event.preventDefault();
    const form = event.currentTarget;
    const url = form.action;

    try {
        const formData = new FormData(form);
        const responseData = await postFormDataAsJson({
            url,
            formData
        });
        console.log({
            responseData
        });
        next(responseData)
    } catch (error) {
        console.error(error);
        alert(error.message);
    }
}

async function removeByID(url, id) {
    let response = await fetchJSONData(url.replace("{id}", id), {
        method: 'DELETE',
        headers: {
            'Content-Type': 'application/json',
        },
        credentials: 'include',
    })
    updateStatusBox(url, response)
}

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
