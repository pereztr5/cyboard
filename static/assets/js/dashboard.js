$('#flag-form').on('click', '#flag-submit', submitFlag);
$('#flag-form').on('keypress', '#flag-value', function(e) {
    if (e.which == 13) {
        submitFlag();
        return false;
    }
});

$('input#flag-value').keypress(function(e) {
    if (e.which == 13) {
        $('#flag-submit').submit();
        return false;
    }
});

function submitFlag() {
    var flagValue = $('#flag-value').val();
    var challengeValue = $('#flag-submit').data('challenge');

    if (flagValue.length > 0) {
        $.ajax({
            url: '/challenge/verify/all',
            type: 'POST',
            dataType: 'html',
            data: {
                flag: flagValue
            },
            success: function(value) {
                if (value == '0') {
                    $('#flag-enter').removeClass('has-error').addClass('has-success');
                    $('#flag-enter .glyphicon').removeClass('glyphicon-remove').addClass('glyphicon-ok');
                } else if (value == '1') {
                    $('#flag-enter').removeClass('has-success').addClass('has-error');
                    $('#flag-enter .glyphicon').removeClass('glyphicon-ok').addClass('glyphicon-remove');
                } else if (value == '2') {
                    $('#flag-enter').removeClass('has-success has-error').addClass('has-warning');
                    $('#flag-enter .glyphicon').removeClass('glyphicon-ok glyphicon-remove').addClass('glyphicon-warning-sign');
                }
            },
        });
    }
}
