const mobileUserAgent = /Android.*Mobile|iPhone|iPod|Windows Phone|IEMobile|Opera Mini|BlackBerry|webOS/i;

// The mobile朋友圈 layout intentionally follows the browser's user agent.
// This lets a phone browser requesting the desktop site keep the desktop UI.
export const useMobileUserAgent = () => {
  const isMobileUserAgent = useState<boolean>("mobileUserAgent", () => false);

  if (import.meta.client) {
    isMobileUserAgent.value = mobileUserAgent.test(navigator.userAgent);
  }

  return isMobileUserAgent;
};
