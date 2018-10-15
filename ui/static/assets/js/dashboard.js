$('#flag-form').on('click', '#flag-submit', submitFlag);
$('#flag-form').on('keypress', '#flag-value', function (e) {
    if (e.which === 13) {
        submitFlag();
        return false;
    }
});

function submitFlag() {
    var flagValue = $('#flag-value').val();
    var challengeValue = $('#flag-submit').data('challenge');

    if (flagValue.length > 0) {
        $.ajax({
            url: '/api/blue/challenges',
            type: 'POST',
            dataType: 'html',
            data: {
                flag: flagValue
            },
            success: function(value) {
                var flag_enter = $('#flag-enter');
                var glyphicon = flag_enter.find('.glyphicon');
                if (value === '0') {
                    flag_enter.removeClass('has-error').addClass('has-success');
                    glyphicon.removeClass('glyphicon-remove').addClass('glyphicon-ok');
                } else if (value === '1') {
                    flag_enter.removeClass('has-success').addClass('has-error');
                    glyphicon.removeClass('glyphicon-ok').addClass('glyphicon-remove');
                } else if (value === '2') {
                    flag_enter.removeClass('has-success has-error').addClass('has-warning');
                    glyphicon.removeClass('glyphicon-ok glyphicon-remove').addClass('glyphicon-warning-sign');
                }
            },
        });
    }
}
