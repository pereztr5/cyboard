const $teamsCapFlags = $(".teams-captured-flags")
    , $tabs = $teamsCapFlags.find('.nav-tabs');

/* Do some dancing to plop the tabs in their nav */
$teamsCapFlags.find('.tab-content a.nav-link')
    .parent().detach().appendTo($tabs);

$tabs.on('click', 'a', function(e) {
    e.preventDefault();
    $(this).tab('show');
});

$tabs.find('li:first-child a').tab('show');

