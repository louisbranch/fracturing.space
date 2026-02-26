function appCurrentPath(pathname) {
	if (!pathname) {
		return window.location.pathname || "/";
	}
	if (typeof pathname === "object") {
		if (pathname instanceof URL && pathname.pathname) {
			return pathname.pathname;
		}
		if (typeof pathname.pathname === "string") {
			return pathname.pathname;
		}
		pathname = String(pathname);
	}
	if (typeof pathname !== "string" || pathname.charAt(0) !== "/") {
		try {
			return new URL(pathname, window.location.origin).pathname;
		} catch (err) {
			return window.location.pathname || "/";
		}
	}
	return pathname;
}

function syncAppNavActiveLinks(currentPath) {
	currentPath = appCurrentPath(currentPath);
	var links = document.querySelectorAll("a[data-nav-item]");
	if (!links.length) {
		return;
	}
	links.forEach(function(link) {
		var href = link.getAttribute("href");
		link.classList.remove("active");
		link.classList.remove("menu-active");
		link.classList.remove("btn-active");
		if (!href) {
			return;
		}
		var target;
		try {
			target = new URL(href, window.location.origin);
		} catch (err) {
			return;
		}
		var shouldBeActive = currentPath === target.pathname;
		if (!shouldBeActive && target.pathname !== "/") {
			shouldBeActive = currentPath.indexOf(target.pathname + "/") === 0;
		}

		if (link.classList.contains("btn")) {
			link.classList.toggle("btn-active", shouldBeActive);
		} else {
			link.classList.toggle("menu-active", shouldBeActive);
		}
	});
}

// TODO(web2-routepath): source this from server route metadata so app-shell behavior stays aligned with routepath ownership.
var campaignWorkspaceRoutePattern = /^\/app\/campaigns\/(?!create(?:\/|$))[^/]+(?:\/|$)/;
var campaignMainCoverClass = "w-full max-w-none px-0 pt-20 flex-1";
var fallbackMainClass = "w-full max-w-none px-4 pt-20 flex-1";

function mainFallbackClass(main) {
	if (!main) {
		return fallbackMainClass;
	}
	var fromDom = main.getAttribute("data-fallback-class");
	if (fromDom) {
		return fromDom;
	}
	return fallbackMainClass;
}

function mainDefaultClass(main) {
	if (!main) {
		return fallbackMainClass;
	}
	var fromDom = main.getAttribute("data-default-class");
	if (fromDom) {
		return fromDom;
	}
	return mainFallbackClass(main);
}

function isCampaignWorkspaceRoute(pathname) {
	return campaignWorkspaceRoutePattern.test(appCurrentPath(pathname));
}

function appMainContainerClass(mainStyle, extraClass, defaultClass, main) {
	var baseClass = defaultClass || mainFallbackClass(main);
	if (mainStyle) {
		baseClass = campaignMainCoverClass;
	}
	if (extraClass) {
		baseClass = baseClass + " " + extraClass;
	}
	return baseClass;
}

function appMainMetadata(main) {
	var metadata = main.querySelector("[data-app-main-style]");
	if (!metadata) {
		return null;
	}

	return {
		style: metadata.getAttribute("data-app-main-style") || "",
		extraClass: metadata.getAttribute("data-app-main-extra-class") || "",
	};
}

function isCampaignMainStyle(style) {
	return (
		typeof style === "string" &&
		style.indexOf("background-image: linear-gradient(to bottom") === 0
	);
}

function syncCampaignMainState(main) {
	var metadata = appMainMetadata(main);
	var style = "";
	var extraClass = "";
	var defaultClass = mainDefaultClass(main);

	if (metadata) {
		style = metadata.style;
		extraClass = metadata.extraClass;
	}
	if (!style) {
		style = main.getAttribute("data-default-style") || "";
	}

	main.setAttribute("style", style);
	main.className = appMainContainerClass(style, extraClass, defaultClass, main);
}

function syncMainStateForRoute(currentPath) {
	var main = document.getElementById("main");
	if (!main) {
		return;
	}

	if (isCampaignWorkspaceRoute(currentPath)) {
		syncCampaignMainState(main);
		return;
	}

	var defaultStyle = main.getAttribute("data-default-style") || "";
	var defaultClass = mainDefaultClass(main);

	if (isCampaignMainStyle(defaultStyle)) {
		main.removeAttribute("style");
		defaultClass = mainFallbackClass(main);
	} else {
		if (defaultStyle) {
			main.setAttribute("style", defaultStyle);
		} else {
			main.removeAttribute("style");
		}
	}

	if (main.className !== defaultClass) {
		main.className = defaultClass;
	}
}

function appPathFromHtmxDetail(detail) {
	if (!detail) {
		return null;
	}
	if (detail.pathInfo && detail.pathInfo.requestPath) {
		return detail.pathInfo.requestPath;
	}
	if (detail.requestConfig && detail.requestConfig.path) {
		return detail.requestConfig.path;
	}
	if (detail.elt && typeof detail.elt.getAttribute === "function") {
		var href = detail.elt.getAttribute("href") || detail.elt.getAttribute("hx-get");
		if (href) {
			return href;
		}
	}
	if (detail.xhr && detail.xhr.responseURL) {
		return detail.xhr.responseURL;
	}
	return null;
}

function syncAppChromeState(currentPath) {
	syncAppNavActiveLinks(currentPath);
	syncMainStateForRoute(currentPath);
}

syncAppChromeState();
document.addEventListener("DOMContentLoaded", function() {
	syncAppChromeState();
});
window.addEventListener("popstate", function() {
	syncAppChromeState();
});
document.addEventListener("htmx:beforeSwap", function(event) {
	if (event.detail && event.detail.xhr && event.detail.xhr.status === 200) {
		window.scrollTo({top: 0, behavior: "instant"});
	}
	if (
		event.detail &&
		event.detail.xhr &&
		event.detail.xhr.status >= 400 &&
		event.detail.target &&
		event.detail.target.id === "main"
	) {
		event.detail.shouldSwap = true;
		event.detail.isError = false;
		window.scrollTo({top: 0, behavior: "instant"});
	}
	var nextPath = appPathFromHtmxDetail(event && event.detail);
	if (isCampaignWorkspaceRoute(nextPath)) {
		return;
	}
	syncAppChromeState(nextPath);
});
document.addEventListener("htmx:afterSwap", function(event) {
	syncAppChromeState(appPathFromHtmxDetail(event && event.detail));
});
document.addEventListener("htmx:afterSettle", function(event) {
	syncAppChromeState(appPathFromHtmxDetail(event && event.detail));
});
