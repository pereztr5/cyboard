const $bonusAwardPanel = $('.bonus-award-panel');


$bonusAwardPanel.children('form').on('submit', function submitBonusPoints(event) {
    event.preventDefault();
    const $form = $(this);
    const findInput = (name) => $form.find(`input[name=${name}]`);

    // Parse form data into JSON
    const data = {};
    ["points", "reason"].forEach(field => {
        data[field] = findInput(field).val();
    });
    data.points = parseInt(data.points, 10);
    data.teams = $form.find(`select[name=teams]`).val().map(pts => parseInt(pts, 10));

    const url = `/api/admin/grant_bonus`;
    ajaxJSON('POST', url, data).done(() => {
        $form.trigger('reset');
        const msg = `${data.teams.length} teams awarded '${data.points}' points!`;
        alert(msg);
    }).fail((xhr) => {
        alert(getXhrErr(xhr));
    });
});
