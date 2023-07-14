import { version } from './version'
import { onLCP, onFID, onCLS } from 'web-vitals';

var view = Math.random().toString(36).substring(2, 12);
var hadActivity = false;
var scrollPerc = 0;

var reportURL = "";

function sendClick(target) {
    SendMetric({ "Event": "click", "Target": target, "SessionId": view })
}

function notifyClick(e) {
    sendClick(e.target.id);
}
function notifyChange(e) {
    SendMetric({ "Event": "click", "Target": e.target.id, "Value": e.target.value, "SessionId": view })
}

function onMouse(e) {
    hadActivity = true;
}

function onScroll(e) {
    hadActivity = true;
    var perc = Math.round((window.scrollY / document.body.scrollHeight) * 100);
    if (perc != scrollPerc) {
        scrollPerc = perc;
    }
}

function reportActivity() {
    if (hadActivity) {
        hadActivity = false;
        SendMetric({
            "Event": "activity",
            "SessionId": view,
            "ScrollPerc": scrollPerc.toString(),
        });
    }
}

function addElementHandlers(className, eventName, handler) {
    var elements = document.getElementsByClassName(className);
    for (var i = 0; i < elements.length; i++) {
        elements[i].addEventListener(eventName, handler);
    }
}

function recordVital({ name, value, id, navigationType }) {
    SendMetric({
        "Event": "vitals", "SessionId": view,
        [name]: value, "navigationType": navigationType
    });
}

export async function SendMetric(data) {
    if (reportURL == "") {
        return;
    }
    // Fire and forget, don't care about response or success.
    fetch(reportURL, {
        body: JSON.stringify(data),
        method: 'POST',
        referrerPolicy: "no-referrer-when-downgrade",
        keepalive: true,
    }).then(function (response) {
        if (!response.ok) {
            console.log("Failed to send metric to " + reportURL + ": " + response.statusText)
        }
    })
    .catch(error => console.log("Failed to send metric to " + reportURL + ": " + error));
}

export function SetupMetrics(url, reportIntervalSecs) {
    reportURL = url;
    // Log the load via a performance observer to get pageload time.
    const observer = new PerformanceObserver((list) => {
        list.getEntries().forEach((entry) => {
            if (entry.duration == 0) {
                return;
            }
            SendMetric({
                "Event": "pageview", "SessionId": view,
                "Page": document.URL, "Referer": document.referrer,
                "LoadTime": entry.duration, "JSVersion": version
            })
            // Don't need more timing after we get the pageview.
            observer.disconnect();
        });
    });
    observer.observe({ type: "navigation", buffered: true });
    // Watch for events
    addElementHandlers("notify-click", "click", notifyClick);
    addElementHandlers("notify-change", "change", notifyChange);
    // Watch for activity
    document.addEventListener("scroll", onScroll);
    document.addEventListener("mousemove", onMouse);
    if (reportIntervalSecs < 20) {
        reportIntervalSecs = 20;
    }
    setInterval(reportActivity, reportIntervalSecs * 1000);
    // Record web vitals
    onCLS(recordVital);
    onFID(recordVital);
    onLCP(recordVital);
}