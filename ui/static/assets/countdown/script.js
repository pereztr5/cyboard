function getTimeRemaining(endtime) {
  const t = Date.parse(endtime) - Date.parse(new Date());
  return {
    'total':   t,
    'days':    Math.floor( t / (1000  * 60  * 60   * 24)),
    'hours':   Math.floor((t / (1000  * 60  * 60)) % 24),
    'minutes': Math.floor((t /  1000  / 60) % 60),
    'seconds': Math.floor((t /  1000) % 60),
  };
}

function initializeClock(elementID, endtime) {
  const clock = document.getElementById(elementID);

  function updateClock() {
    const t = getTimeRemaining(endtime);

    clock.querySelector('.days').innerHTML    = t.days;
    clock.querySelector('.hours').innerHTML   = ('0' + t.hours).slice(-2);
    clock.querySelector('.minutes').innerHTML = ('0' + t.minutes).slice(-2);
    clock.querySelector('.seconds').innerHTML = ('0' + t.seconds).slice(-2);

    if (t.total <= 0) {
      clearInterval(timeinterval);
      const wait = Math.floor(Math.random() * (5 * 1000));
      window.setTimeout(() => window.location.reload(), wait);
    }
  }

  updateClock();
  const timeinterval = setInterval(updateClock, 1000);
}

const deadline = new Date(new Date().getTime() + (_countdown_duration * 1000));

initializeClock('countdown', deadline);
