// Instagram 原寸画像オープナー - background service worker

// 個別ポスト（/p/、/reel/、/{user}/p/ など）のページでのみメニューを出す
const POST_URL_PATTERNS = [
  "https://www.instagram.com/p/*",
  "https://www.instagram.com/reel/*",
  "https://www.instagram.com/reels/*",
  "https://www.instagram.com/tv/*",
  "https://www.instagram.com/*/p/*",
  "https://www.instagram.com/*/reel/*"
];

const MENU_ALL = "open-all-images";
const MENU_CURRENT = "open-current-image";

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: MENU_ALL,
      title: "このポストの画像をすべて原寸で開く",
      contexts: ["all"],
      documentUrlPatterns: POST_URL_PATTERNS
    });
    chrome.contextMenus.create({
      id: MENU_CURRENT,
      title: "表示中の画像だけを原寸で開く",
      contexts: ["all"],
      documentUrlPatterns: POST_URL_PATTERNS
    });
  });
});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (!tab || tab.id === undefined) return;
  if (info.menuItemId !== MENU_ALL && info.menuItemId !== MENU_CURRENT) return;

  let result;
  try {
    const [injection] = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: collectOriginalImageUrls,
      args: [info.menuItemId === MENU_CURRENT]
    });
    result = injection && injection.result;
  } catch (e) {
    console.error("スクリプトの実行に失敗しました", e);
    return;
  }

  if (!result || result.error) {
    await showAlert(tab.id, (result && result.error) || "画像を取得できませんでした。");
    return;
  }

  // 元のタブの右隣に順番に開く
  let index = tab.index;
  for (const url of result.urls) {
    index += 1;
    await chrome.tabs.create({ url, index, active: false, openerTabId: tab.id });
  }
});

function showAlert(tabId, message) {
  return chrome.scripting
    .executeScript({
      target: { tabId },
      func: (msg) => alert("[Instagram 原寸画像オープナー]\n" + msg),
      args: [message]
    })
    .catch(() => {});
}

/**
 * ページ内で実行される関数。
 * ※ executeScript で注入されるため、外部の変数・関数は参照できない（自己完結させること）。
 * 戻り値: { urls: string[] } または { error: string }
 */
async function collectOriginalImageUrls(currentOnly) {
  const ALPHABET =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";

  const m = location.pathname.match(/\/(?:p|reel|reels|tv)\/([A-Za-z0-9_-]+)/);
  if (!m) return { error: "ポストの URL ではありません。" };
  const shortcode = m[1];

  // カルーセルで表示中の画像番号（Instagram は ?img_index=N を URL に付ける。1 始まり）
  const imgIndexParam = parseInt(new URLSearchParams(location.search).get("img_index"), 10);
  const currentIndex = Number.isFinite(imgIndexParam) && imgIndexParam > 0 ? imgIndexParam - 1 : 0;

  // ショートコード -> メディア ID
  let mediaId = 0n;
  for (const ch of shortcode.slice(0, 11)) {
    const v = ALPHABET.indexOf(ch);
    if (v < 0) return { error: "ショートコードを解析できませんでした。" };
    mediaId = mediaId * 64n + BigInt(v);
  }

  // 候補の中から最大サイズの画像 URL を選ぶ
  const pickLargest = (item) => {
    const cands = (item && item.image_versions2 && item.image_versions2.candidates) || [];
    let best = null;
    for (const c of cands) {
      if (!best || c.width * c.height > best.width * best.height) best = c;
    }
    return best ? best.url : null;
  };

  // 1) Instagram の内部 API から取得（ログイン中のセッションを使用）
  try {
    const res = await fetch(`/api/v1/media/${mediaId}/info/`, {
      credentials: "include",
      headers: {
        "X-IG-App-ID": "936619743392459",
        "X-Requested-With": "XMLHttpRequest"
      }
    });
    if (res.ok) {
      const data = await res.json();
      const item = data && data.items && data.items[0];
      if (item) {
        let medias = item.carousel_media && item.carousel_media.length ? item.carousel_media : [item];
        if (currentOnly) {
          medias = [medias[Math.min(currentIndex, medias.length - 1)]];
        }
        // media_type: 1 = 画像, 2 = 動画, 8 = カルーセル。動画は除外
        const urls = medias
          .filter((x) => x && x.media_type !== 2)
          .map(pickLargest)
          .filter(Boolean);
        if (urls.length) return { urls };
        return { error: "このポスト（または表示中のスライド）には画像がありません（動画のみ）。" };
      }
    }
  } catch (e) {
    // フォールバックへ
  }

  // 2) フォールバック: ページ上の <img> の srcset から最大サイズを取得
  //    （未ログイン時など。カルーセルは読み込み済みのスライドのみ取得可能）
  const root = document.querySelector("article") || document.querySelector("main") || document;
  const seen = new Set();
  const urls = [];
  for (const img of root.querySelectorAll("img")) {
    const rect = img.getBoundingClientRect();
    // アイコンなどの小さい画像は除外
    if (Math.max(img.naturalWidth, rect.width) < 300) continue;
    let bestUrl = img.currentSrc || img.src;
    let bestW = img.naturalWidth || 0;
    for (const part of (img.srcset || "").split(",")) {
      const [u, w] = part.trim().split(/\s+/);
      const width = parseInt(w, 10);
      if (u && width > bestW) {
        bestW = width;
        bestUrl = u;
      }
    }
    if (!bestUrl || seen.has(bestUrl)) continue;
    seen.add(bestUrl);
    if (currentOnly) {
      // 画面中央に最も近い画像を「表示中」とみなす
      const cx = rect.left + rect.width / 2;
      const inView = rect.width > 0 && cx > 0 && cx < window.innerWidth;
      if (!inView) continue;
    }
    urls.push(bestUrl);
  }
  if (currentOnly && urls.length > 1) urls.length = 1;
  if (urls.length) return { urls };
  return { error: "画像が見つかりませんでした。ページを再読み込みしてからお試しください。" };
}
