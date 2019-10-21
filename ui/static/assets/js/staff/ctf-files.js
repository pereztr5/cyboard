const $filesModal = $('#ctf-file-modal')
    , $filelist = $filesModal.find('.filelist tbody');

/* Construct an individual <tr> for the file listings.
 * Returns a dangling <tr> DOM element.
 *
 * Function is curried, because I was feeling spicy.
 *
 * Param f is a FileInfo JSON from the API, with the structure:
 * { name: String, size: Int, mod_time: <Date-String> } */
function buildFileTableRow(url) {
    return f => {
        const escapedFilename = window.encodeURIComponent(f.name);
        const fileURL = `${url}/${escapedFilename}`;
        const bytes = niceBytes(f.size);
        const timestamp = new Date(f.mod_time).toUTCString();

        return $(`<tr class="flag-file" data-name="${f.name}" />`)
            .append($(`<td class="text-center" />`).append(`<i class="fa fa-download" />`))
            .append($(`<td />`).append($(`<a class="btn-block" />`).attr('href', fileURL).text(f.name)))
            .append($(`<td />`).text(bytes))
            .append($(`<td />`).text(timestamp))
            .append($(`<th class="text-right" />`)
                .append($(`<button class="btn btn-sm btn-danger btn-file-delete" />`)
                    .append(`<i class="fa fa-trash" />`)));
    }
}

/* Show file list modal */
$ctfConfig.on('click', '.btn-files', function showFlagEditorModal(event) {
    const $row = getCtfRow($(event.currentTarget));
    const flagID = $row.data('flag-id');
    const name = $row.children().eq(1).text();

    $filesModal.find('.modal-title').text(`Files for "${name}"`);
    $filesModal.find('input[name=id]').val(flagID);
    $filesModal.find('input[type=file]').val("");

    const url = `/api/ctf/flags/${flagID}/files`;
    $.getJSON(url).done(files => {
        $filelist.empty().append(files.map(buildFileTableRow(url)));
    }).fail((xhr) => {
        $filelist.empty();
        if (!xhr.responseText.includes("no such file or directory")) {
            alert(`Failed to fetch files for ${name}: ${getXhrErr(xhr)}`);
        }
    }).always(() => {
        $filesModal.modal('show');
    });
});

/* Add files */
$filesModal.find('form').on('submit', function uploadFlagFiles(event) {
    event.preventDefault();
    const $form = $(this);
    const data = new FormData(this);

    const loading = toggleLoadingButton($(this).find('button[type=submit]'));
    loading(true);

    const flagID = $form.siblings('input[name=id]').val();
    const url = `/api/ctf/flags/${flagID}/files`;

    // There's some quirks here to make jQuery.ajax work with file uploads.
    $.ajax({
        url,
        data,
        method: "POST",
        contentType: false,
        processData: false,
    }).done(() => {
        loading(false);

        // Append new files to listing
        const now = new Date();
        const $fileInput = $filesModal.find('input[type=file]');
        const fileInfos = $.map($fileInput.get(0).files, f => {
            const {name, lastModified, size} = f;
            return {name, size, mod_time: now};
        });
        $filelist.append(fileInfos.map(buildFileTableRow(url)));

        // Clear file input
        $fileInput.val("");
    }).fail((xhr) => {
        loading(false);
        alert(getXhrErr(xhr));
    });
});


/* Delete a file */
$filelist.on('click', '.btn-file-delete', function deleteFlagFile(event) {
    const $btn = $(event.currentTarget);
    const $row = getCtfRow($btn);

    // GET and DELETE use the same URL, so pull from the download link
    const url = $row.find('a').attr('href');
    const filename = $row.data('name');

    if(confirm(`Are you sure you want to delete "${filename}"`)) {
        ajaxJSON('DELETE', url).done(() => {
            $row.remove();
        }).fail((xhr) => {
            alert(getXhrErr(xhr));
        });
    };
});

