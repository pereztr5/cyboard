/* NOTE: This code is very similar to teams.js, and could be refactored. */

// Global DOM references to the teams table and editor modal.
const $modal = $('#service-edit-modal');
const $cfgTable = $('.services-config-table');

const getButtonTableRow = ($btn) => $btn.parentsUntil('tr').parent();

// pad any date component with a leading 0. e.g. Sept -> 09
function pad(datePart) {
    return ("0" + datePart).slice(-2);
}

/* Show modal editor */
$cfgTable.on('click', '.btn-edit', function showServiceEditorModal(event) {
    const $row = $(event.currentTarget).parentsUntil('tr').parent()
    const id = $row.data('service-id');

    const $form = $modal.find('form');
    const findInput = (name) => $form.find(`input[name=${name}]`);

    // Query the service api directly to get the data for this service in JSON
    const url = `/api/admin/services/${id}`
    $.getJSON(url).done(srv => {
        // Set a bunch of form fields from the JSON
        ["id","name","description","total_points","script"].forEach(k => {
            findInput(k).val(srv[k]);
        });

        // Quote all args because there's only one text input and this is
        // the safest way to maintain the hacky arg parsing.
        findInput("args").val(srv.args.map(s => `"${s}"`).join(" "));
        findInput("disabled").prop('checked', srv.disabled);

        // Decompose start time into two separate inputs, because between
        // browsers this is the most supported way to create a date+time picker.
        const dt = new Date(srv.starts_at);
        findInput("starts_at_date").val(
            `${dt.getFullYear()}-${pad(dt.getMonth()+1)}-${pad(dt.getDate())}`);
        findInput("starts_at_time").val(
            `${pad(dt.getHours())}:${pad(dt.getMinutes())}`);

        $modal.find('.modal-title').text(`Edit "${srv.name}"`);

        $modal.modal('show');
    }).fail(xhr => {
        alert(`Failed to get service "${id}": ${getXhrErr(xhr)}`);
    });
});

/* When showing the modal, hide the "Delete" button if the modal is being used to create. */
$modal.on('show.bs.modal', function(event) {
    const isNewService = $(event.relatedTarget).hasClass("btn-add-service");
    $modal.find('form').find('.delete-service').toggleClass("hidden", isNewService);
});


/* Parse the modal form into either a Service POST or PUT for the server. */
function serviceFormAsJson($form) {
    const findInput = (name) => $form.find(`input[name=${name}], select[name=${name}]`);
    const strInput = (name) => findInput(name).val();
    const intInput = (name) => parseInt(findInput(name).val(), 10);
    const floatInput = (name) => parseFloat(findInput(name).val(), 10);

    // Parse form data into JSON
    const data = {
        id: intInput("id"),
        name: strInput("name"),
        description: strInput("description"),
        total_points: floatInput("total_points"),
        script: strInput("script"),
        args: splitArgs(strInput("args")),
        disabled: findInput("disabled").prop("checked"),
    };

    // Combine the date+time picker values together.
    // Browser handles conversion from local time picked by user -> to UTC.
    const starts_at_date = findInput("starts_at_date").val();
    const starts_at_time = findInput("starts_at_time").val();
    data.starts_at = new Date(`${starts_at_date}T${starts_at_time}`);

    return data;
}

/* Submit new/editted service */
$modal.find('form').on('submit', function saveService(event) {
    event.preventDefault();
    const $form = $(this);
    const data = serviceFormAsJson($form);

    const isNewService = data.id === -1;
    if(isNewService) {
        delete data.id;
        const url = `/api/admin/services`;
        ajaxAndReload('POST', url, data, `${data.name} created!`);
    } else {
        const url = `/api/admin/services/${data.id}`;
        ajaxAndReload('PUT', url, data, `${data.name} updated!`);
    }
});

/* Delete Team */
$modal.find('form').on('click', '.delete-service', function deleteService(event) {
    const $form = $(event.delegateTarget);
    const id = $form.find("input[name=id]").val();
    const name = $form.find("input[name=name]").val();

    BootstrapDialog.confirm(`Are you sure you want to delete "${name}"`, yes => {
        if(!yes) { return; }

        const url = `/api/admin/services/${id}`;
        ajaxJSON('DELETE', url).done(() => {
            $cfgTable.children('tbody').find(`[data-service-id=${id}]`).remove();
            $modal.modal('hide');
        }).fail((xhr) => {
            alert(getXhrErr(xhr));
        });
    });
});


/* Add new service, button below the table */
$('.btn-add-service').on('click', function showTeamAddModal(event) {
    const $form = $modal.find('form');
    $form.trigger('reset');

    $modal.find('.modal-title').text("Add new team");
    $modal.find('input[name=id]').val("-1");

    // Modal show is wired up in the HTML, which then exposes the "relatedTarget"
    // on the event (part of bootstrap's API), which is used to hide/show a Delete button.
    //$modal.modal('show');
});

