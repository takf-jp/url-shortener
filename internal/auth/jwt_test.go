package auth
// 正常系: 発行→検証で UserID が取れる
// 期限切れ: 過去日付の Token で Verify が失敗
// 改ざん: ペイロードを変更すると Verify が失敗
// 別アルゴリズム: 不正なアルゴリズムを拒否
