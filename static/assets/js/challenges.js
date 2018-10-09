// DOM references for the global modal and inner flag submission form fields
const $modal = $('#flag-modal')
    , $title  = $modal.find('.flag-modal-title')
    , $points = $modal.find('.flag-modal-points')
    , $desc   = $modal.find('.flag-description')
    , $files  = $modal.find('.flag-file-list')

    , $flag   = $modal.find('input[name=flag]')
    , $name   = $modal.find('input[name=name]')
    , $id     = $modal.find('input[name=id]');

// Global reference to button that opened the modal
let $btn;

// Modal display, fetches extra info from the server
$('.challenge-list').on('click', 'button', function(event) {
    $btn = $(event.currentTarget);
    const flagID = $btn.data('flag-id');
    const [name, category, points] = $btn.find('p').map(function() { return $(this).text(); }).get();

    if (flagID === $id.val()) {
        // Minimal caching
        $modal.modal('show');
        return;
    }

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
                $(`<a class="col-md-4 col-sm-6 col-xs-12 text-truncate" />`).attr('href', `${fileURL}/${f.name}`)
                    .append($(`<div class="btn btn-block btn-primary text-overflow" />`)
                        .append(`<span class="fa fa-download" />`)
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
    $('input:visible:enabled:first', this).focus();
})

// When the modal goes away, clean up
$modal.on('hidden.bs.modal', function(event) {
    $flag.val('');
    $modal.find('.alert').finish().hide();
});


function markAsSolved($btn) {
    $btn.addClass('negate')
        .prepend(`<span class="fa fa-check-square" />`);
}

// Flag submission
$modal.find('form').on('submit', function(event) {
    event.preventDefault();
    const $form = $(this);
    const $alert = $modal.find('.alert');

    $alert.stop(true, true); // Flush animations

    const setStatus = (cls, text) => {
        $alert.attr('class', `alert ${cls}`);
        $alert.text(text);
    };

    const inputVal = inputName => $form.find(`input[name='${inputName}']`).val();

    $.post('/api/blue/challenges', {
        challenge: inputVal('name'),
        flag: inputVal('flag'),
    }).done(flagState => {
        switch(flagState) {
        case 0:  setStatus('alert-success', 'You got it!'); markAsSolved($btn); break;
        case 1:  setStatus('alert-danger', 'Incorrect'); break;
        case 2:  setStatus('alert-warning', 'You already solved this'); break;
        default: setStatus('alert-danger', '...Something weird happened'); break;
        }
    }).fail(r => {
        const msg = r.status === 0 ? 'Failed to connect to server. Is your internet working?' : r.responseText;
        setStatus('alert-danger', msg);
    }).always(() => {
        $alert.slideDown(300).delay(1400).slideUp(300);

        const $submit = $form.find('button[type=submit]');
        $submit.prop('disabled', true);
        setTimeout(() => $submit.prop('disabled', false), 1000);
    });
});

