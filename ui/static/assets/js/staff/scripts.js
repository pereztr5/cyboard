const $modal = $('#scripts-run-modal')
    , $form  = $modal.find('form');

const formInput = (name) => $form.find(`input[name=${name}]`);

$('tbody').on('click', '.btn-script-run', function(event) {
    const $btn = $(event.currentTarget);
    const $row = $btn.parentsUntil('tr').parent();
    const scriptName = $row.data('name');

    $modal.find(".modal-title").text(`Test Run "${scriptName}"`);
    formInput("name").val(scriptName);
    formInput("args").val('');
    $form.find("pre").hide();

    $modal.modal('show');
});

$form.on('submit', function(event) {
    event.preventDefault();
    const name = formInput("name").val();
    const args = splitArgs(formInput("args").val());

    const $btn = $form.find('button[type=submit]');
    const loading = toggleLoadingButton($btn);
    loading(true);

    const url = `/api/admin/scripts/${name}/run`;
    ajaxJSON('POST', url, args).done((message) => {
        const $out = $form.find('pre');
        $out.text(message);
        $out.slideDown(200);
    }).fail((xhr) => {
        alert(getXhrErr(xhr));
    }).always(() => {
        loading(false);
    });
});
