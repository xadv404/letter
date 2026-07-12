(function () {
  var toggle = document.querySelector(".menu-toggle");
  var sidebar = document.querySelector(".sidebar");

  if (toggle && sidebar) {
    toggle.addEventListener("click", function () {
      sidebar.classList.toggle("open");
    });

    document.addEventListener("click", function (e) {
      if (
        sidebar.classList.contains("open") &&
        !sidebar.contains(e.target) &&
        !toggle.contains(e.target)
      ) {
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
