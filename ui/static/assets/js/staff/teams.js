// Global DOM references to the teams table and editor modal.
// The modal gets reused for both adding a single new team and editing an existing one.
const $modal = $('#team-edit-modal');
const $cfgTable = $('.team-config-table');

const getButtonTableRow = ($btn) => $btn.parentsUntil('tr').parent();

// isBlueteam helps determine if the blueteam_ip form/json field is necessary
// (when displaying the modal or submitting JSON data).
const isBlueteam = (role_name) => role_name === "blueteam";

/* Show modal editor */
$cfgTable.on('click', '.btn-edit', function showTeamEditorModal(event) {
    const $row = $(event.currentTarget).parentsUntil('tr').parent()

    // Very tight DOM coupling
    const $form = $modal.find('form');
    const findInput = (name) => $form.find(`input[name=${name}]`);

    const $cells = $row.children();
    const cellText = (idx) => $cells.eq(idx).text();

    $modal.find('.modal-title').text(`Edit "${cellText(1)}"`);

    // Plug in the role name to the combo box, and fire a change event to enable/disable
    // the blueteam_ip field, if editing a blueteam.
    const role_name = cellText(2);
    $form.find("select[name=role_name]").val(role_name).trigger('change');

    findInput("id").val(cellText(0));
    findInput("name").val(cellText(1));
    findInput("blueteam_ip").val(cellText(3));

    const isDisabled = $cells.eq(4).children().length > 0;
    findInput("disabled").prop('checked', isDisabled);

    // Always reset the password field.
    findInput("password").val('');

    $modal.modal('show');
});

/* Disable blueteam_ip field based on selected role_name */
$modal.find('form').on('change', 'select[name=role_name]', function(event) {
    const $form = $(event.delegateTarget);
    const role_name = $(event.currentTarget).val();

    const enable = isBlueteam(role_name);
    $form.find("input[name=blueteam_ip]")
        .prop("required", enable)
        .parent(".form-group").toggle(enable);
});

/* When showing the modal, hide the "Delete" button if the modal is being used to create. */
$modal.on('show.bs.modal', function(event) {
    const isNewTeam = $(event.relatedTarget).hasClass("btn-add-team");
    $modal.find('form').find('.delete-team').toggle(!isNewTeam);
});


/* Parse the modal form into either a TeamModRequest for the server. */
function teamFormAsJson($form) {
    const findInput = (name) => $form.find(`input[name=${name}], select[name=${name}]`);
    const strInput = (name) => findInput(name).val();
    const intInput = (name) => parseInt(findInput(name).val(), 10);

    // Parse form data into JSON
    const data = {
        id: intInput("id"),
        name: strInput("name"),
        role_name: strInput("role_name"),
        disabled: findInput("disabled").prop("checked"),
    };

    // Blueteams have one extra field, for their significant IP octet.
    if(isBlueteam(data.role_name)) {
        data.blueteam_ip = intInput("blueteam_ip");
    }

    // "password" field should only be included if it's meant to change.
    // Empty passwords are rejected by the server.
    const pass = strInput("password");
    if(pass !== "") {
        data.password = pass;
    }

    return data;
}

/* Submit new/editted Team */
$modal.find('form').on('submit', function saveTeam(event) {
    event.preventDefault();
    const $form = $(this);
    const data = teamFormAsJson($form);

    const isNewTeam = data.id === -1;
    if(isNewTeam) {
        delete data.id;
        const url = `/api/admin/teams`;
        ajaxAndReload('POST', url, data, `${data.name} created!`);
    } else {
        const url = `/api/admin/teams/${data.id}`;
        ajaxAndReload('PUT', url, data, `${data.name} updated!`);
    }
});

/* Delete Team */
$modal.find('form').on('click', '.delete-team', function deleteTeam(event) {
    const $form = $(event.delegateTarget);
    const id = $form.find("input[name=id]").val();
    const name = $form.find("input[name=name]").val();

    if(confirm(`Are you sure you want to delete "${name}"`)) {
        const url = `/api/admin/teams/${id}`;
        ajaxJSON('DELETE', url).then(() => {
            $('.team-config-table > tbody').find(`[data-team-id=${id}]`).remove();
            $modal.modal('hide');
        }).catch((xhr) => {
            alert(getXhrErr(xhr));
        });
    };
});


/* Add new team, button below the table */
$('.btn-add-team').on('click', function showTeamAddModal(event) {
    const $form = $modal.find('form');
    $form.trigger('reset');
    $form.find("select[name=role_name]").trigger("change");

    $modal.find('.modal-title').text("Add new team");
    $modal.find('input[name=id]').val("-1");

    // Modal show is wired up in the HTML, which then exposes the "relatedTarget"
    // on the event (part of bootstrap's API), which is used to hide/show a Delete button.
    //$modal.modal('show');
});


/* Upload Blue teams via CSV */
const $csvUploadForm = $('.csv-upload form');

(function initTeamsCsvEditor() {
    const placeholderCsv = `
                    Name, IP, Password
             red jaguars,  1, smokestack
          91st tech unit,  2, bologna
            joel spolsky,  3, xmas_monkEE
"a single sleepy, doxin",  4, d2hhdCB0aGUgZnVjayBkaWQgeW91IGV4cGVjdAo=
`.slice(1, -1); // trim the newlines at the start and end (but not the spaces)

    $csvUploadForm.find('textarea').val(placeholderCsv);
})();

$csvUploadForm.on('submit', function addTeamsViaCSV(event) {
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
    data = data.map((row, idx) => {
        row = Object.assign(...Object.entries(row).map(([k, v]) => ({[k.toLowerCase()]: v })));

        try {
            row.blueteam_ip = parseInt(row.ip, 10);
            delete row.ip;
        } catch(e) {
            alert(`row ${idx}: parsing IP as int failed: ${e.message}`);
            throw e;
        }

        return row;
    });

    // This request can take a while, because each team's password has to be hashed,
    // so a spinner will be displayed to show some form of progress
    const loading = toggleLoadingButton($(this).find('button[type=submit]'));
    loading(true);

    const url = `/api/admin/blueteams`;
    ajaxJSON("POST", url, data).then(() => {
        alert(`${data.length} teams created! Page will reload.`);
        window.location.reload();
    }).catch((xhr) => {
        alert(getXhrErr(xhr));
    }).always(() => {
        loading(false);
    });
});

