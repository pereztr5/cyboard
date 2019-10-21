// DOM references for the global modal and inner flag submission form fields
const $modal = $('#flag-modal')
    , $title  = $modal.find('.flag-modal-title')
    , $points = $modal.find('.flag-modal-points')
    , $desc   = $modal.find('.flag-description')
    , $files  = $modal.find('.flag-file-list')

    , $form   = $modal.find('form')
    , $flag   = $modal.find('input[name=flag]')
    , $name   = $modal.find('input[name=name]')
    , $id     = $modal.find('input[name=id]');

// Global reference to button that opened the modal
let $btn;

// Modal display, fetches extra info from the server
$('.challenge-list').on('click', 'button', function(event) {
    $btn = $(event.currentTarget);
    const flagID = $btn.data('flag-id');
    const [name, points] = $btn.find('p').map(function() { return $(this).text(); }).get();

    if (flagID === $id.val()) {
        // Minimal caching
        $modal.modal('show');
        return;
    }

    $form.trigger('reset');

    $title.text(name);
    $points.text(points);

    $id.val(flagID);
    $name.val(name);

    const fileURL = `/api/blue/challenges/${flagID}/files`;

    const qs = [
        $.get(`/api/blue/challenges/${flagID}`).then(desc => {
            $desc.html(marked(desc));
        }, () => {
            $desc.html(`<p class="text-warning">!! Couldn't fetch challenge description for some reason !!</p>`);
        }).promise(),

        $.getJSON(fileURL).then(fileList => {
            $files.empty().append(fileList.map(f =>
                $(`<a class="col-md-4 col-sm-6 text-truncate" />`).attr('href', `${fileURL}/${window.encodeURIComponent(f.name)}`)
                    .append($(`<div class="btn btn-block btn-primary text-truncate" />`)
                        .append(`<span class="fa fa-download mr-1" />`)
                        .append($(`<small />`).text(f.name)))
            ));
        }, () => {
            console.warn("Couldn't get files for "+name);
            $files.empty();
        }).promise(),
    ];

    $.when(...qs).always(() => { $modal.modal('show'); });
});

// When the modal shows up, focus the submission box
$modal.on('shown.bs.modal', function(event) {
    $('input:visible:enabled:first', this).trigger('focus');
})

// When the modal goes away, clean up
$modal.on('hidden.bs.modal', function(event) {
    $modal.find('.alert').finish().hide();
});

// Flag submission
$modal.find('form').on('submit', function(event) {
    event.preventDefault();
    const $form = $(this);
    const $alert = $modal.find('.alert');

    handleFlagSubmission($form, $alert, {alert_delay: 1400, $flagCard: $btn});
});

