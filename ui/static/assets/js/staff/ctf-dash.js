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

// Count the total number of solves.
// 100% reliable.
function sumIntColumn(acc, td) {
    return acc + parseInt(td.textContent, 10);
}
const $flagSubCounts = $('table.most-submitted-flag');
const totalSolves = $flagSubCounts.find('td:nth-child(4)').toArray().reduce(sumIntColumn, 0);
$flagSubCounts.siblings('h6').append(
    $(`<small class="text-muted">`).text(`- ${totalSolves} total solves`));
