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

var appNavbarSelector = "[data-app-navbar]";
var appScrollOffsetVariable = "--app-scroll-offset";
var fallbackAppNavbarHeightPx = 80;
var appScrollOffsetGapPx = 16;
var campaignWorkspaceRouteArea = "campaign-workspace";
var campaignMainCoverClass = "w-full max-w-none px-0 pt-20 flex-1";
var fallbackMainClass = "w-full max-w-none px-4 pt-20 flex-1";

function syncAppScrollOffset() {
	var root = document.documentElement;
	if (!root || !root.style || typeof root.style.setProperty !== "function") {
		return;
	}

	var navbar = document.querySelector(appNavbarSelector);
	var navbarHeight = fallbackAppNavbarHeightPx;
	if (navbar && typeof navbar.getBoundingClientRect === "function") {
		var measuredHeight = Math.ceil(navbar.getBoundingClientRect().height);
		if (Number.isFinite(measuredHeight) && measuredHeight > 0) {
			navbarHeight = measuredHeight;
		}
	}

	root.style.setProperty(appScrollOffsetVariable, String(navbarHeight + appScrollOffsetGapPx) + "px");
}

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

function appMainRouteArea(main) {
	if (!main) {
		return "";
	}
	var routeArea = main.getAttribute("data-app-route-area");
	return routeArea ? String(routeArea).trim() : "";
}

function isCampaignWorkspaceArea(routeArea) {
	return routeArea === campaignWorkspaceRouteArea;
}

function isCampaignWorkspaceMetadata(metadata) {
	return metadata && metadata.routeArea && isCampaignWorkspaceArea(metadata.routeArea);
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
	var routeArea = appMainRouteArea(main);
	var metadata = main.querySelector("[data-app-main-style]");
	if (!metadata) {
		return routeArea ? {
			routeArea: routeArea,
			style: main.getAttribute("data-default-style") || "",
			backgroundPreviewURL: main.getAttribute("data-default-background-preview") || "",
			backgroundFullURL: main.getAttribute("data-default-background-full") || "",
		} : null;
	}

	var metadataRouteArea = metadata.getAttribute("data-app-route-area");
	return {
		style: metadata.getAttribute("data-app-main-style") || "",
		extraClass: metadata.getAttribute("data-app-main-extra-class") || "",
		backgroundPreviewURL: metadata.getAttribute("data-app-main-background-preview") || "",
		backgroundFullURL: metadata.getAttribute("data-app-main-background-full") || "",
		routeArea: metadataRouteArea ? String(metadataRouteArea).trim() : routeArea,
	};
}

function appMainBackgroundStyle(backgroundURL) {
	if (!backgroundURL) {
		return "";
	}
	return (
		"background-image: url(\"" + backgroundURL + "\"); " +
		"background-size: cover; background-position: center; background-repeat: no-repeat;"
	);
}

function composeMainStyle(baseStyle, backgroundURL) {
	baseStyle = baseStyle || "";
	var backgroundStyle = appMainBackgroundStyle(backgroundURL);
	if (!backgroundStyle) {
		return baseStyle;
	}
	if (!baseStyle) {
		return backgroundStyle;
	}
	if (baseStyle.charAt(baseStyle.length - 1) === ";") {
		return baseStyle + " " + backgroundStyle;
	}
	return baseStyle + "; " + backgroundStyle;
}

function shouldUseCampaignWorkspaceStyle(metadata) {
	if (!metadata) {
		return false;
	}
	if (isCampaignWorkspaceMetadata(metadata)) {
		return true;
	}
	return !!(metadata.backgroundPreviewURL || metadata.backgroundFullURL);
}

var prewarmedImageURLs = Object.create(null);

function prewarmImageURL(url) {
	url = typeof url === "string" ? url.trim() : "";
	if (!url || prewarmedImageURLs[url]) {
		return;
	}
	prewarmedImageURLs[url] = true;
	var img = new Image();
	img.decoding = "async";
	img.src = url;
}

function prewarmDeclaredImages(root) {
	var scope = root || document;
	if (!scope || typeof scope.querySelectorAll !== "function") {
		return;
	}
	scope.querySelectorAll("[data-image-prefetch-url]").forEach(function(node) {
		prewarmImageURL(node.getAttribute("data-image-prefetch-url") || "");
	});
}

function preloadBackgroundImage(url, onReady) {
	url = typeof url === "string" ? url.trim() : "";
	if (!url) {
		if (typeof onReady === "function") {
			onReady("");
		}
		return;
	}
	var img = new Image();
	var settled = false;
	function resolve(resolvedURL) {
		if (settled) {
			return;
		}
		settled = true;
		if (typeof onReady === "function") {
			onReady(resolvedURL);
		}
	}
	img.decoding = "async";
	img.onload = function() {
		resolve(url);
	};
	img.onerror = function() {
		resolve("");
	};
	img.src = url;
	if (typeof img.decode === "function") {
		img.decode().then(function() {
			resolve(url);
		}).catch(function() {
			// Let onload/onerror resolve the request.
		});
	}
}

function syncCampaignMainState(main, metadata) {
	metadata = metadata || appMainMetadata(main);
	var baseStyle = "";
	var extraClass = "";
	var defaultClass = mainDefaultClass(main);
	var previewURL = "";
	var fullURL = "";

	if (metadata) {
		baseStyle = metadata.style;
		extraClass = metadata.extraClass;
		previewURL = metadata.backgroundPreviewURL;
		fullURL = metadata.backgroundFullURL;
	}
	if (!baseStyle) {
		baseStyle = main.getAttribute("data-default-style") || "";
	}
	if (!previewURL) {
		previewURL = main.getAttribute("data-default-background-preview") || "";
	}
	if (!fullURL) {
		fullURL = main.getAttribute("data-default-background-full") || "";
	}

	var initialBackgroundURL = previewURL || fullURL || "";
	main.setAttribute("style", composeMainStyle(baseStyle, initialBackgroundURL));
	main.className = appMainContainerClass(composeMainStyle(baseStyle, initialBackgroundURL), extraClass, defaultClass, main);

	if (!fullURL || fullURL === initialBackgroundURL) {
		main.removeAttribute("data-main-background-token");
		return;
	}

	var token = fullURL + "::" + String(Date.now());
	main.setAttribute("data-main-background-token", token);
	preloadBackgroundImage(fullURL, function(resolvedURL) {
		if (!resolvedURL) {
			return;
		}
		if (main.getAttribute("data-main-background-token") !== token) {
			return;
		}
		main.setAttribute("style", composeMainStyle(baseStyle, resolvedURL));
		main.className = appMainContainerClass(composeMainStyle(baseStyle, resolvedURL), extraClass, defaultClass, main);
	});
}

function syncMainStateForRoute() {
	var main = document.getElementById("main");
	if (!main) {
		return;
	}
	var metadata = appMainMetadata(main);

	if (shouldUseCampaignWorkspaceStyle(metadata)) {
		syncCampaignMainState(main, metadata);
		return;
	}

	var defaultStyle = main.getAttribute("data-default-style") || "";
	var defaultClass = mainDefaultClass(main);
	var defaultPreviewURL = main.getAttribute("data-default-background-preview") || "";
	var defaultFullURL = main.getAttribute("data-default-background-full") || "";
	var hasDefaultBackground = !!(defaultPreviewURL || defaultFullURL);
	if (hasDefaultBackground) {
		defaultClass = mainFallbackClass(main);
	}

	main.removeAttribute("data-main-background-token");
	if (hasDefaultBackground) {
		if (defaultStyle) {
			main.setAttribute("style", defaultStyle);
		} else {
			main.removeAttribute("style");
		}
	} else if (defaultStyle || defaultPreviewURL || defaultFullURL) {
		main.setAttribute("style", composeMainStyle(defaultStyle, defaultPreviewURL || defaultFullURL || ""));
	} else {
		main.removeAttribute("style");
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
	syncAppScrollOffset();
	syncAppNavActiveLinks(currentPath);
	syncMainStateForRoute(currentPath);
	initAppToasts();
}

var defaultToastHideAfterMs = 4500;

function toastHideAfterMs(toast) {
	if (!toast || typeof toast.getAttribute !== "function") {
		return defaultToastHideAfterMs;
	}
	var raw = Number.parseInt(toast.getAttribute("data-app-toast-hide-after-ms"), 10);
	if (!Number.isFinite(raw) || raw < 0) {
		return defaultToastHideAfterMs;
	}
	return raw;
}

function dismissAppToast(toast) {
	if (!toast) {
		return;
	}
	if (toast.getAttribute("data-app-toast-dismissed") === "true") {
		return;
	}
	toast.setAttribute("data-app-toast-dismissed", "true");
	toast.classList.add("app-toast-exit");
	window.setTimeout(function() {
		if (toast.parentNode) {
			toast.parentNode.removeChild(toast);
		}
		var stack = document.getElementById("app-toast-stack");
		if (stack && stack.children.length === 0 && stack.parentNode) {
			stack.parentNode.removeChild(stack);
		}
	}, 220);
}

function initAppToasts() {
	var toasts = document.querySelectorAll("[data-app-toast='true']");
	if (!toasts.length) {
		return;
	}
	toasts.forEach(function(toast) {
		if (toast.getAttribute("data-app-toast-initialized") === "true") {
			return;
		}
		toast.setAttribute("data-app-toast-initialized", "true");
		window.setTimeout(function() {
			dismissAppToast(toast);
		}, toastHideAfterMs(toast));
	});
}

function convertLocalTimes() {
	document.querySelectorAll("time[data-app-localtime]").forEach(function(el) {
		var iso = el.getAttribute("datetime");
		if (!iso) return;
		var date = new Date(iso);
		if (isNaN(date.getTime())) return;
		var now = new Date();
		var delta = now - date;
		// Only convert if >= 7 days (relative times are already timezone-agnostic)
		if (delta >= 7 * 24 * 60 * 60 * 1000) {
			el.textContent = date.toLocaleString(undefined, {
				year: "numeric", month: "short", day: "numeric",
				hour: "2-digit", minute: "2-digit"
			});
		}
	});
}

syncAppChromeState();
convertLocalTimes();
prewarmDeclaredImages(document);
document.addEventListener("DOMContentLoaded", function() {
	syncAppChromeState();
	prewarmDeclaredImages(document);
});
window.addEventListener("resize", syncAppScrollOffset);
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
});
document.addEventListener("htmx:afterSwap", function(event) {
	syncAppChromeState(appPathFromHtmxDetail(event && event.detail));
	prewarmDeclaredImages(event && event.detail ? event.detail.target : document);
});
document.addEventListener("htmx:afterSettle", function(event) {
	syncAppChromeState(appPathFromHtmxDetail(event && event.detail));
	convertLocalTimes();
	prewarmDeclaredImages(event && event.detail ? event.detail.target : document);
});
