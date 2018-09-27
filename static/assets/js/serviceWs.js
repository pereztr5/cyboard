$(function() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const endpoint = `${protocol}//${window.location.host}/api/public/services/live`;
    const conn = new WebSocket(endpoint);

    conn.onclose = (evt) => {
        // reuse the top-left (i)nfo box, replace it with a warning message
        const $errorNode = $('.sv-help .fa-info-circle')
            .attr('class', 'sv-help-error text-warning')
            .text('Live feed closed. Page should try to reload soon.');
    };
    conn.onmessage = (evt) => {
        results = JSON.parse(evt.data);
        try {
            syncServices(results)
        } catch(e) {
            if (e instanceof ErrorTeamSync) {
                conn.close();
                const sometimeInTenSecs = Math.floor(Math.random() * (10 * 1000));
                console.warn(`Page reloading in ${sometimeInTenSecs}ms`);
                window.setTimeout(() => window.location.reload(), sometimeInTenSecs);
            } else {
                conn.close();
                console.error(e);
            }
        }
    };
});

function ErrorTeamSync() {
    this.message = 'teams out of sync';
}

function syncServices(data) {
    /* data looks like: [{
        "service_id":1,
        "service_name":"WWW Content",
        "statuses": ["partial", "pass", ...],
     },
     {...}, ...]
    */

    /* Diff the dom, apply changes */

    const $serviceDisplay = $('.service-statuses');
    const $rows = $serviceDisplay.children('.sv-row');
    const $teamRows = $rows.first().children('.sq-team');
    const $serviceRows = $rows.slice(1);

    // If the number of columns is out of sync because a team was added/disabled (unlikely),
    // just reload the page by throwing up an exception.
    if (data.length === 0 || data[0].statuses.length !== $teamRows.length) {
        throw new ErrorTeamSync();
    }

    let i, j;
    for (i=j=0; i < $serviceRows.length && j < data.length;) {
        const {service_id, service_name, statuses} = data[j];

        const $sRow = $($serviceRows[i]);
        const domServiceID = $sRow.data('check');

        if (domServiceID > service_id) {
            // insert service row
            const $newService = newServiceRow(service_id, service_name, statuses);
            $sRow.before($newService);
            j++;
        } else if (domServiceID < service_id) {
            // delete service row
            $sRow.remove();
            i++;
        } else {
            // update service row
            updateServiceStatusBoxes($sRow, statuses);
            i++; j++;
        }
    }

    /* Handle trails in each array; At most one of these two loops will run. */

    for (; i < $serviceRows.length; i++) {
        const $sRow = $($serviceRows[i]);
        $sRow.remove();
    }

    for (; j < data.length; j++) {
        const {service_id, service_name, statuses} = data[j];
        $serviceDisplay.append(
            newServiceRow(service_id, service_name, statuses)
        );
    }
}

function newServiceRow(id, name, statuses) {
    const $row = $(`<div class="sv-row"></div>`).attr('data-check', id);
    $row.append( $(`<div class="sq sq-label sq-service"></div>`).text(name) );

    const $boxes = statuses.map((_, idx) => {
        const $ico = $(`<span class="fa" aria-hidden="true"></span>`).attr('data-status', '');
        const $sq = $(`<div class="sq"></div>`).append($ico);
        return $sq;
    });
    $row.append($boxes);
    updateServiceStatusBoxes($row, statuses);
    return $row;
}

function updateServiceStatusBoxes($serviceRow, statuses) {
    const $statusBoxes = $serviceRow.find("span[data-status]");
    statuses.forEach((status, idx) => {
        const $statusBox = $($statusBoxes[idx]);

        // If the status hasn't changed, skip past this box/node.
        if($statusBox.attr('data-status') === status) {
            return;
        }

        let newIcon;
        switch(status) {
        case 'pass': newIcon = 'fa-arrow-circle-up text-success'; break;
        case 'fail': newIcon = 'fa-arrow-circle-down text-danger blink'; break;
        case 'partial': newIcon = 'fa-exclamation-circle text-warning'; break;
        default:     newIcon = 'fa-question-circle text-muted'; break;
        }
        $statusBox.attr('class', `fa ${newIcon}`)
                  .attr('data-status', status);
    });
}
