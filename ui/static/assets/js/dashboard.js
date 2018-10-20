const $hiddenFlagForm = $('.hidden-flag-form')
    , $btn = $hiddenFlagForm.find('button[type=submit]');

$hiddenFlagForm.on('submit', function(event) {
    event.preventDefault();
    const $alert = $hiddenFlagForm.find('.alert');

    handleFlagSubmission($hiddenFlagForm, $alert, {
        alert_delay: 2700, anonymous: true
    });
});

