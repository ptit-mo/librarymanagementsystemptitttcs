function fetchData({ url, fetchOptions, next }) {
    console.log("fetchData", url, fetchOptions, next)
    fetch(url, fetchOptions)
        .then(response => {
            if (!response.ok) {
                if (response.status === 401) {
                    throw new Error("Unauthorized")
                }
            }
            return response.json()
        }
        )
        .then(data => {
            console.log(data)
            if (data["message"] === true) {
                alert(data["message"])
            }
            updateStatusBox(url, data)
            if (next != null) {
                next(data)
            }
        })
        .catch(error => {
            if (error.message === "Unauthorized") {
                window.location.href = "/fe/login.html"
            }
            throw error
        })
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
    console.log(data); // Handle response data
    setTimeout(() => {
        statusBox.style.display = "none";
    }, 5000)
}

const generatedFormIDPrefix = 'generatedAddForm-';
function generateAddForm(
    itemName = 'book',
    fields = [{ name: 'title', type: 'text', label: 'Title', optional: true }],
    submitEndpoint = '/admin/book',
) {
    // add html content
    const formContainer = document.getElementById("addContainer");
    formContainer.innerHTML = '';
    const previewContainer = document.getElementById("addPreview");
    previewContainer.innerHTML = '';
    const form = document.createElement('form');
    const preview = document.createElement('div');
    formGroups = ""
    previewGroups = ""
    id = `${generatedFormIDPrefix}${itemName}`
    fields.forEach(field => {
        formGroups += makeFormElement(field)
        previewGroups += `
            <p><strong>${field.label}:</strong> <span id="${id}-${field.name}preview"></span></p>
        `
    })
    form.innerHTML = `
        <form class="container" id="${id}">
            ${formGroups}
            <input type="submit" value="Add new ${itemName}" >
        </form>
        `
    preview.innerHTML = `
        <div class="preview" id="${id}preview">
            <h3>Preview</h3>
            ${previewGroups}
        </div>
    `;
    formContainer.appendChild(form);
    previewContainer.appendChild(preview);

    // update preview on input
    form.addEventListener('input', function () {
        fields.forEach(field => {
            if (field.type === 'file') {
                document.getElementById(`${id}-${field.name}file`).addEventListener('input', function () {
                    const file = this.files[0];
                    const reader = new FileReader();
                    reader.onload = function (e) {
                        document.getElementById(`${id}-${field.name}preview`).innerHTML = `<img style="display: block; max-width: 300px; max-height: 200px; width: auto; height: auto" src="${e.target.result}" alt="Uploaded image" />`;
                    }
                    reader.readAsDataURL(file);
                });
                return
            }
            const value = document.getElementById(`${id}-${field.name}`).value;
            document.getElementById(`${id}-${field.name}preview`).textContent = value;
        });
    });

    form.addEventListener('submit', function (e) {
        e.preventDefault()
        let payload = {}

        const uploadJobs = []
        fields.map(field => {
            // upload image
            if (field.type === 'file') {
                const fileInput = document.getElementById(`${id}-${field.name}file`);
                const file = fileInput.files[0];
                if (!file) {
                    // console.error('No file selected');
                    alert('No file selected');
                }

                const formData = new FormData();
                formData.append('file', file);
                uploadJobs.push(fetch('/admin/uploadimage', {
                    method: 'POST',
                    credentials: 'include',
                    body: formData,
                }).then(response => {
                    if (!response.ok) {
                        alert('Failed to upload image');
                        throw new Error('Failed to upload image');
                    }
                    return response.json();
                }).then(data => {
                    return { key: field.name, value: data['path'] }
                }).catch(error => {
                    console.error('There was a problem handling data:', error);
                    throw error;
                }));
            } else {
                // collect other values
                let value = document.getElementById(`${id}-${field.name}`).value;
                if (field.type == 'number') {
                    value = parseInt(value)
                }
                payload[field.name] = value;
            }
        })
        // submit form with payload if all images are uploaded
        Promise
            .all(uploadJobs)
            .then((uploadedKeyPaths) => {
                uploadedKeyPaths.forEach(uploadedKeyPath => {
                    if (uploadedKeyPath) {
                        payload[uploadedKeyPath.key] = uploadedKeyPath.value;
                    }
                });
                console.log(payload)
                fetchData({
                    url: submitEndpoint,
                    fetchOptions: {
                        method: 'POST',
                        credentials: 'include',
                        body: JSON.stringify(payload),
                        headers: { 'Content-Type': 'application/json', }
                    }, next: (data) => {

                    }
                })
            }).catch(error => {
                alert('There was a problem uploading files. Please try again later');
                throw error;
            })

    });
}

function makeFormElement(field) {
    switch (field.type) {
        case 'select':
            return makeSelectFormElement(field)
        case 'file':
            return makeFileFormElement(field)
        default:
            return makeDefaultFormElement(field)
    }
}

function makeSelectFormElement(field) {
    return `
    <div class="form-group">
    <label class="input-group-text" for="${field.name}">${field.label}:</label>
    <select id="${id}-${field.name}" name="${field.name}">
    ${field.selectOptions.map(option => `<option value="${option}">${option}</option>`).join('\n')}
    </select>
    </div>
    `
}

function makeFileFormElement(field) {
    return `
    <div class="input-group mb-3">
            <label class="input-group-text" for="${field.name}">${field.label}:</label>
            <input type="file" class="form-control" id="${id}-${field.name}file" name="${field.name}" accept="image/*" ${field.optional ? '' : 'required'}>
    </div>
    `
}

function makeDefaultFormElement(field) {
    return `
    <div class="form-group">
            <label class="input-group-text" for="${field.name}">${field.label}:</label>
            <input type="${field.type}" class="form-control"  id="${id}-${field.name}" name="${field.name}" required>
    </div>
    `
}

function generateNavBar() {
    const navBar = document.getElementById('navbar');
    navBar.innerHTML = `
    <div class="container-fluid">
        <nav class="collapse navbar-collapse bs-navbar-collapse">
            <ul class="nav navbar-nav nav-pills">
                <li>
                    <a href="/fe/index.html">Home</a>
                </li>
                <li>
                    <a href="/fe/login.html">Login</a>
                </li>
                <li>
                    <a href="/fe/book.html">Book</a>
                </li>
                <li>
                    <a href="/fe/user.html">User</a>
                </li>
                <li>
                    <a href="/fe/borrow_history.html">Borrow History</a>
                </li>
                <li>
                    <a href="/fe/logout.html">Logout</a>
                </li>
        </nav>
    </div>
</header>
    `;
}

let nlimit = 10
function generateListForm(
    fields = ['id', 'title', 'author', 'type', 'quantity', 'cover'],
    submitEndpoint = '/internal/books?',
) {
    // get offset from input id offset or default to 0 then fill to input
    let offset = document.getElementById('offset').value || 0
    // add table header
    let tableHeaders = fields.map(field => `<th id="${field}-header">${field}</th>`).join('\n')
    const table = document.getElementById('list');
    table.innerHTML = `
        <thead>
            ${tableHeaders}
        </thead>
    `

    let tableRows = ""
    // get all items then add table body
    fetchData({
        url: submitEndpoint + `offset=${offset}&limit=${nlimit}`,
        fetchOptions: {
            method: 'GET',
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
            }
        }, next: (data) => {
            console.log(data); // Handle response data
            if (data == undefined) {
                data = []
            }
            data.forEach(item => {
                tableRows += `
                <tr>
                    ${fields.map(field => isValidImageUrl(String(item[field])) ?
                    `<td><img style="display: block; max-width: 100px; max-height: 100px; width: auto; height: auto" src="${item[field]}" alt="${field}" /></td>` :
                    `<td>${item[field]}</td>`).join('\n')}
                </tr>
                `
            })
            table.innerHTML += `
                <tbody>
                    ${tableRows}
                </tbody>
            `
            if (data.length == nlimit) {
                document.getElementById('offset').value = data[data.length - 1].id
            } else {
                document.getElementById('offset').value = 0
            }
        }
    })
}

function isValidImageUrl(uri) {
    return uri.match(/https?.*\.(jpeg|jpg|gif|png)$/) != null;
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
    formData
}) => {
    const plainFormData = Object.fromEntries(formData.entries());
    const formDataJsonString = JSON.stringify(plainFormData);

    const fetchOptions = {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
        },
        body: formDataJsonString,
    };


    // alert("about to post" + formDataJsonString)
    const response = await fetch(url, fetchOptions);

    if (!response.ok) {
        const errorMessage = await response.text();
        throw new Error(errorMessage);
    }

    return response.json();
}
/**
 * Event handler for a form submit event.
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLFormElement/submit_event
 * @example const exampleForm = document.getElementById("example-form");
 *          exampleForm.addEventListener("submit", handleFormSubmit);
 * @param {SubmitEvent} event
 */
var handleFormSubmit = async (event) => {
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
        window.location.href = "/fe/book.html"
    } catch (error) {
        console.error(error);
        alert(error.message);
    }
}
