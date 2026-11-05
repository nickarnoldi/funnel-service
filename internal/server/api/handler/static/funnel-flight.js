window.FunnelFlight = (function () {
	const endpoint = "{{.EndpointURL}}";
	const cookiePrefix = "funnel-flight";
	const cookieMaxAge = 60 * 60 * 24; // 1 day

	const sessionParameters = {};
	const cookieParameters = {};

	const payload = {};

	let initd = false;
	let shouldSend = false;
	let pageLoad = new Date().getTime();
	let stepNumber = 1;

	let fieldRemap = {};

	hydrateLandingPageCookie();

	function discoverPayload() {
        const elements = document.querySelectorAll("input, select, textarea");
        elements.forEach(function (elem) {
            if (elem.name) {
                fieldName = (elem.name in fieldRemap ? fieldRemap[elem.name] : elem.name)
                if (elem.type === "checkbox") {
                    if (elem.checked) {
                        payload[fieldName] = elem.value ?? elem.checked;
                    } else if (!(fieldName in payload)) {
                        payload[fieldName] = false;
                    }
                } else if (elem.type === "radio") {
                    if (elem.checked) {
                        payload[fieldName] = elem.value ?? elem.checked;
                    } else if (!(fieldName in payload)) {
                        payload[fieldName] = "";
                    }
                } else {
                    payload[fieldName] = elem.value;
                }
            }
        });
	}

	function getLandingPage() {
		const landingPage = hydrateLandingPageCookie();
		const landingURL = new URL(landingPage);

		const currentParams = new URLSearchParams(window.location.search);
		currentParams.forEach((value, key) => {
			if (!landingURL.searchParams.has(key)) {
				landingURL.searchParams.set(key, value);
			}
		});

		return landingURL.toString();
	}

	function hydrateParametersFromCookies() {
		const collectableCookies = [
			's1',
			's2',
			's3',
			's4',
			's5',
			'aid',
			'cpid',
			'rid',
			'ci',
			'ck',
			'fbclid',
			'gclid',
			'ttclid',
			'obclid',
			'obclid',
			'tclid'
		]

		for (let cookies = document.cookie.split(/;\s*/), i = cookies.length - 1; i >= 0; i--) {
			const cookie = cookies[i];
			const parts = cookie.split("=");
			const name = parts[0].replace(`${cookiePrefix}.`, "");
			const value = parts.slice(1).join("=");
			if (cookie.indexOf(`${cookiePrefix}.`) === 0) {
				sessionParameters[name] = decodeURIComponent(value);
			} else if (collectableCookies.includes(name)) {
				cookieParameters[name] = decodeURIComponent(value);
			}
		}
	}

	function getCookieValue(name, defaultValue = undefined) {
		for (let cookies = document.cookie.split(/;\s*/), i = cookies.length - 1; i >= 0; i--) {
			if (!cookies[i].indexOf(`${name}=`)) {
				return cookies[i].replace(`${name}=`, "");
			}
		}

		if (defaultValue !== undefined) {
			setCookieValue(name, defaultValue);
		}

		return defaultValue;
	}

	function setCookieValue(name, value) {
		document.cookie = `${name}=${value}; max-age=${cookieMaxAge}; path=/`;
	}

	function hydrateLandingPageCookie() {
        const url = new URL(document.location.href);
        url.pathname = url.pathname.replace(/\/$/, '');

		return getCookieValue(`${cookiePrefix}.landing_page`, url.toString());
	}

	function getSessionLength() {
		return Math.round((new Date().getTime() - pageLoad) / 1000);
	}

	function setupOnClose() {
		document.addEventListener("visibilitychange", () => {
			if (document.visibilityState === "hidden") {
				submitDataOnExit();
			}
		});
	}

	function submitDataOnExit() {
		if (shouldSend) {
			discoverPayload();
			hydrateParametersFromCookies();

			const jsonPayload = JSON.stringify({
				...sessionParameters,
				body: payload,
				cookies: cookieParameters,
				step_number: stepNumber,
				landing_page: getLandingPage(),
				session_length: getSessionLength(),
			})

			if (navigator.sendBeacon) {
				navigator.sendBeacon(endpoint, jsonPayload);
				return
			}

			// Fallback for browsers that do not support sendBeacon
			fetch(endpoint, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: jsonPayload,
				keepalive: true
			})
		}
	}

	return {
		init: function (remapFields = undefined) {
			if (initd) {
				return;
			}
			initd = true;
			shouldSend = true;

			if (remapFields) {
				fieldRemap = remapFields
			}

			setupOnClose();
		},
		enable: function (remapFields = undefined) {
			if (!initd) {
				return this.init(remapFields);
			}
			shouldSend = true;
		},
		disable: function () {
			shouldSend = false;
		},
		nextStep: function () {
			stepNumber += 1;
		},
		previousStep: function () {
			stepNumber -= 1;
		},
		update: function (name, value) {
			payload[name] = value;
		},
		// debug: function () {
		// 	console.log("FunnelFlight || Should send data:", shouldSend, "landingPage", getLandingPage(), "pageTime", getSessionLength(), "Current payload:", payload)
		// }
	}
})();
const FunnelFlight = window.FunnelFlight || {};
