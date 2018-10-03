/* Helper methods */

// Send JSON to the server
function ajaxJSON(method, url, data) {
    return $.ajax({
        url,
        method,
        data: JSON.stringify(data),
        contentType: "application/json; charset=utf-8",
    });
}

const getXhrErr = xhr => xhr.status === 0 ? "Network error!" : xhr.responseText;

// Send JSON via POST/PUT to the API, prompt with the response, and on success reload the page.
function ajaxAndReload(method, url, data, successMsg) {
    return ajaxJSON(method, url, data).done(() => {
        alert(successMsg + " Page will reload.");
        window.location.reload();
    }).fail((xhr) => {
        alert(getXhrErr(xhr));
    });
}

// Humanize byte sizes
/* Credits to Faust on Stackoverflow: https://stackoverflow.com/a/39906526 */
function niceBytes(x){
    const units = ['bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    let l = 0, n = parseInt(x, 10) || 0;
    while(n >= 1024 && ++l)
          n = n/1024;
    return(n.toFixed(n >= 10 || l < 1 ? 1 : 2) + ' ' + units[l]);
}

