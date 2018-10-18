// DOM references for the global modal and inner flag submission form fields
const $modal = $('#ctf-edit-modal')
    , $editor  = $modal.find('textarea')
    , $preview = $modal.find('.preview')
    , $mkdnTabs= $modal.find(".nav-tabs a[data-toggle='tab']");

const $ctfConfig = $('.flag-config-table');

const getCtfRow = ($btn) => $btn.parentsUntil('tr').parent();

/* Display flag */
$ctfConfig.on('click', '.btn-flag', function displayFlag(event) {
    const $btn = $(event.currentTarget);
    const flag = $btn.attr('title');
    const name = getCtfRow($btn).children().eq(1).text();

    // show the flag with the text selected, ready to copy+paste
    window.prompt(`Flag for "${name}"`, flag);
});


/* Show modal editor */
$ctfConfig.on('click', '.btn-edit', function showFlagEditorModal(event) {
    const $row = getCtfRow($(event.currentTarget));
    const flagID = $row.data('flag-id');

    // Very tight DOM coupling
    const $form = $modal.find('form');
    const $cells = $row.children();
    const findInput = (name) => $form.find(`input[name=${name}]`);
    const cellText = (idx) => $cells.eq(idx).text();

    findInput("id").val(cellText(0));
    findInput("name").val(cellText(1));
    findInput("category").val(cellText(2));
    findInput("designer").val(cellText(3));
    findInput("total").val(cellText(4));

    findInput("flag").val($cells.eq(7).find('.btn-flag').attr('title'));
    $modal.find('.modal-title').text(`Edit ${cellText(1)}`);

    const isHidden = $cells.eq(5).children().length > 0;
    findInput("hidden").prop('checked', isHidden);

    // TODO: This fetches the entire challenge again, just
    // to get the description. This is a minor waste.
    $.getJSON(`/api/ctf/flags/${flagID}`).then(chal => {
        $editor.val(chal.body);
        $mkdnTabs.eq(0).tab('show');
    }, (xhr) => {
        $editor.val("");
        $mkdnTabs.eq(1).tab('show');

        $preview.html($(`<p class="text-danger" />`)
                       .text(`!! Couldn't fetch description: ${getXhrErr(xhr)}`));
    })
    .always(() => { $modal.modal('show'); });
});

/* When showing the modal, hide the "Delete" button if the modal is being used to create. */
$modal.on('show.bs.modal', function(event) {
    const isNewChallenge = $(event.relatedTarget).hasClass("btn-add-challenge");
    $modal.find('form').find('.delete-challenge').toggle(!isNewChallenge);
});

/* Live preview of markdown description */
$mkdnTabs.eq(1).on("show.bs.tab", function(event) {
    const desc = $editor.val();
    $preview.html(marked(desc));
});

/* Submit editted Challenge */
$modal.find('form').on('submit', function updateChallenge(event) {
    event.preventDefault();
    const $form = $(event.delegateTarget);
    const findInput = (name) => $form.find(`input[name=${name}]`);

    // Parse form data into JSON
    const data = {};
    data.body = $editor.val();
    data.hidden = findInput("hidden").prop("checked");
    ["id","name","category","designer","flag","total"].forEach(field => {
        data[field] = findInput(field).val();
    });
    data.id = parseInt(data.id, 10);
    data.total = parseFloat(data.total, 10);

    const isNewChallenge = data.id === -1;
    if(isNewChallenge) {
        delete data.id;
        const url = `/api/ctf/new_flag`;
        ajaxAndReload('POSt', url, data, `${data.name} created!`);
    } else {
        const url = `/api/ctf/flags/${data.id}`;
        ajaxAndReload('PUT', url, data, `${data.name} updated!`);
    }
});

/* Add new challenge, button below the table */
$('.btn-add-challenge').on('click', function showChallengeAddModal(event) {
    const $form = $modal.find('form');
    $form.trigger('reset');
    $mkdnTabs.eq(0).tab('show');

    $modal.find('.modal-title').text("Add new challenge");
    $modal.find('input[name=id]').val("-1");

    // Modal show is wired up in the HTML, which then exposes the "relatedTarget"
    // on the event (part of bootstrap's API), which is used to hide/show a Delete button.
    //$modal.modal('show');
});

/* Delete challenge */
$modal.find('form').on('click', '.delete-challenge', function deleteChallenge(event) {
    const $form = $(event.delegateTarget);
    const flagID = $form.find("input[name=id]").val();
    const flagName = $form.find("input[name=name]").val();

    if(confirm(`Are you sure you want to delete "${flagName}"`)) {
        const url = `/api/ctf/flags/${flagID}`;
        ajaxJSON('DELETE', url).done(() => {
            $('.flag-config-table > tbody').find(`[data-flag-id=${flagID}]`).remove();
            $modal.modal('hide');
        }).fail((xhr) => {
            alert(getXhrErr(xhr));
        });
    };
});


/* Upload Challenges via CSV */
const $csvUploadForm = $('.flag-config-csv-upload form');

(function initChallengeCsvEditor() {
    const placeholderFlagCsv = `
     Name,  Category, Designer, Hidden,                   Flag, Points, Description
 crypto-1,       CTF,      Bob,  false,   flag{1337_ch4ll3ng0},    100,  easy peasy
 crypto-2,       CTF,      Bob,   true,  flag{_!@*scr4mbl3r)(},    150,
   prog-1,      Prog,      Bob,  false, flag{why all the work},     50,
  Stage 1,      Wifi,      Dan,  false,          that was easy,     25,
`.slice(1, -1); // trim the newlines at the start and end (but not the spaces)

    $csvUploadForm.find('textarea').val(placeholderFlagCsv);
})();

$csvUploadForm.on('submit', function addChallengesViaCSV(event) {
    event.preventDefault();
    const rawCSV = $csvUploadForm.find('textarea').val();

    let data;
    try {
        data = $.csv.toObjects(rawCSV, { onParseValue: e => e.trimLeft() });
    } catch (e) {
        alert(`CSV Upload failed: ${e.message}`)
        throw e;
    }

    // Lowercase all keys, cast values to scalars, rename fields as needed
    data = data.map(row => {
        row = Object.assign(...Object.entries(row).map(([k, v]) => ({[k.toLowerCase()]: v })));
        row.hidden = row.hidden === "true";
        row.total = parseFloat(row.points);
        delete row.points;
        return row;
    });

    const url = `/api/ctf/flags/`;
    ajaxAndReload('POST', url, data, `${data.length} challenges created!`);
});
