function markAsSolved($btn) {
    $btn.addClass('negate')
        .prepend(`<span class="fa fa-check-square" />`);
}

function handleFlagSubmission($form, $alert, opts) {
    opts = opts || {};
    const alert_delay = opts.alert_delay || 1000;
    const $btn = opts.$flagCard;
    const named_challenge = !opts.anonymous;

    $alert.stop(true, true); // Flush animations

    const setStatus = (cls, text) => {
        $alert.attr('class', `alert ${cls}`);
        $alert.text(text);
    };

    const inputVal = inputName => $form.find(`input[name='${inputName}']`).val();
    const data = { flag: inputVal('flag') };
    if (named_challenge) {
        data.challenge = inputVal('name');
    }

    $.post('/api/blue/challenges', data).then(flagState => {
        switch(flagState) {
        case 0:  setStatus('alert-success', 'You got it!'); if($btn) markAsSolved($btn); break;
        case 1:  setStatus('alert-danger', 'Incorrect'); break;
        case 2:  setStatus('alert-warning', 'You already solved this'); break;
        default: setStatus('alert-danger', '...Something weird happened'); break;
        }
    }).catch(r => {
        let msg = '';
        if (r.status === 0) {
            msg = 'Failed to connect to server. Is your internet working?';
        } else {
            msg = r.responseJSON ? r.responseJSON.status : r.responseText;
        }
        setStatus('alert-danger', msg);
    }).always(() => {
        $alert.slideDown(300).delay(alert_delay).slideUp(300);

        const $submit = $form.find('button[type=submit]');
        $submit.prop('disabled', true);
        setTimeout(() => $submit.prop('disabled', false), alert_delay+300);
    });
};

