(function () {
  var API = "";

  function api(path, opts) {
    opts = opts || {};
    return fetch(API + path, opts).then(function (res) {
      if (!res.ok) {
        return res.text().then(function (t) {
          throw new Error(t || res.statusText);
        });
      }
      if (opts.raw) return res;
      var ct = res.headers.get("content-type") || "";
      if (ct.indexOf("application/json") >= 0) return res.json();
      return res.text();
    });
  }

  window.LetterAPI = {
    getConfig: function () { return api("/api/config"); },
    putConfig: function (cfg) {
      return api("/api/config", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(cfg),
      });
    },
    uploadDomains: function (file) {
      var fd = new FormData();
      fd.append("file", file);
      return api("/api/domains", { method: "POST", body: fd });
    },
    start: function () { return api("/api/start", { method: "POST" }); },
    pause: function () { return api("/api/pause", { method: "POST" }); },
    resume: function () { return api("/api/resume", { method: "POST" }); },
    stop: function () { return api("/api/stop", { method: "POST" }); },
    getState: function () { return api("/api/state"); },
    connectEvents: function (onEvent) {
      var es = new EventSource(API + "/api/events");
      es.onmessage = function (ev) {
        try { onEvent(JSON.parse(ev.data)); } catch (e) {}
      };
      return es;
    },
  };

  var toggle = document.querySelector(".menu-toggle");
  var sidebar = document.querySelector(".sidebar");
  if (toggle && sidebar) {
    toggle.addEventListener("click", function () {
      sidebar.classList.toggle("open");
    });
    document.addEventListener("click", function (e) {
      if (sidebar.classList.contains("open") && !sidebar.contains(e.target) && !toggle.contains(e.target)) {
        sidebar.classList.remove("open");
      }
    });
  }

  var path = window.location.pathname.split("/").pop() || "index.html";
  document.querySelectorAll(".nav-link").forEach(function (link) {
    var href = link.getAttribute("href");
    if (href === path || (path === "" && href === "index.html")) {
      link.classList.add("active");
    }
  });
})();
