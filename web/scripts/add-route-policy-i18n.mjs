/**
 * One-off script: add route_policy.* and 智能路由策略 keys to all locale files.
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const localesDir = path.join(__dirname, '../src/i18n/locales');

const zhCN = {
  智能路由策略: '智能路由策略',
  'route_policy.add_override': '添加覆盖规则',
  'route_policy.channel_name': '渠道名称',
  'route_policy.channels': '个渠道',
  'route_policy.custom_mode_hint': '当前使用自定义路由模式。',
  'route_policy.delete_failed': '删除失败',
  'route_policy.description':
    '配置您账号下的模型 API 智能路由策略。未单独配置时跟随系统默认。',
  'route_policy.disabled': '跟随系统默认',
  'route_policy.disabled_hint':
    '使用管理员在智能路由策略中配置的全局模式与权重。',
  'route_policy.enabled': '启用',
  'route_policy.global_override': '全局',
  'route_policy.global_weight': '系统默认权重',
  'route_policy.group_key': '目标归类',
  'route_policy.group_key_placeholder': '例如 gpt-4o',
  'route_policy.group_models': '包含模型',
  'route_policy.load_failed': '加载路由策略失败',
  'route_policy.load_failed_with_reason': '加载路由策略失败：{{reason}}',
  'route_policy.mode_default': '默认（本站原生）',
  'route_policy.mode_default_hint':
    '我的请求不走 TokenFactory 智能路由，由本站原生逻辑选路。',
  'route_policy.mode_desc': '选择模型 API 请求的渠道选择方式',
  'route_policy.mode_label': '路由模式',
  'route_policy.mode_price': '价格优模式',
  'route_policy.mode_price_hint': '同一归类下按最终单价升序选择渠道。',
  'route_policy.mode_reset': '已恢复跟随系统默认',
  'route_policy.mode_updated': '路由模式已更新',
  'route_policy.mode_weight': '权重模式',
  'route_policy.mode_weight_hint':
    '按我为各归类配置的渠道权重排序；未配置时回退到系统默认权重。',
  'route_policy.model_groups': '模型归类',
  'route_policy.models': '个模型',
  'route_policy.my_override': '我的',
  'route_policy.override_added': '覆盖规则已添加',
  'route_policy.override_deleted': '覆盖规则已删除',
  'route_policy.override_required': '请填写原始模型名与目标归类',
  'route_policy.overrides': '模型归类覆盖',
  'route_policy.overrides_desc':
    '将特定原始模型名映射到指定归类，优先于自动归类规则。',
  'route_policy.price': '单价',
  'route_policy.price_per_1k': '{{symbol}}{{price}}/1K',
  'route_policy.provider': '供应商',
  'route_policy.raw_model': '原始模型名',
  'route_policy.raw_model_placeholder': '例如 gpt-4o-2024-08-06',
  'route_policy.retry': '重试',
  'route_policy.save_failed': '保存失败',
  'route_policy.scope': '范围',
  'route_policy.site_global_ref': '当前站点全局模式',
  'route_policy.title': '智能路由策略',
  'route_policy.user_weight': '我的权重',
  'route_policy.weight_deleted': '权重配置已删除',
  'route_policy.weight_updated': '权重配置已保存',
};

const zhTW = {
  智能路由策略: '智能路由策略',
  'route_policy.add_override': '新增覆蓋規則',
  'route_policy.channel_name': '渠道名稱',
  'route_policy.channels': '個渠道',
  'route_policy.custom_mode_hint': '目前使用自訂路由模式。',
  'route_policy.delete_failed': '刪除失敗',
  'route_policy.description':
    '設定您帳號下的模型 API 智能路由策略。未單獨設定時跟隨系統預設。',
  'route_policy.disabled': '跟隨系統預設',
  'route_policy.disabled_hint':
    '使用管理員在智能路由策略中設定的全域模式與權重。',
  'route_policy.enabled': '啟用',
  'route_policy.global_override': '全域',
  'route_policy.global_weight': '系統預設權重',
  'route_policy.group_key': '目標歸類',
  'route_policy.group_key_placeholder': '例如 gpt-4o',
  'route_policy.group_models': '包含模型',
  'route_policy.load_failed': '載入路由策略失敗',
  'route_policy.load_failed_with_reason': '載入路由策略失敗：{{reason}}',
  'route_policy.mode_default': '預設（本站原生）',
  'route_policy.mode_default_hint':
    '我的請求不走 TokenFactory 智能路由，由本站原生邏輯選路。',
  'route_policy.mode_desc': '選擇模型 API 請求的渠道選擇方式',
  'route_policy.mode_label': '路由模式',
  'route_policy.mode_price': '價格優模式',
  'route_policy.mode_price_hint': '同一歸類下按最終單價升序選擇渠道。',
  'route_policy.mode_reset': '已恢復跟隨系統預設',
  'route_policy.mode_updated': '路由模式已更新',
  'route_policy.mode_weight': '權重模式',
  'route_policy.mode_weight_hint':
    '依我為各歸類設定的渠道權重排序；未設定時回退到系統預設權重。',
  'route_policy.model_groups': '模型歸類',
  'route_policy.models': '個模型',
  'route_policy.my_override': '我的',
  'route_policy.override_added': '覆蓋規則已新增',
  'route_policy.override_deleted': '覆蓋規則已刪除',
  'route_policy.override_required': '請填寫原始模型名與目標歸類',
  'route_policy.overrides': '模型歸類覆蓋',
  'route_policy.overrides_desc':
    '將特定原始模型名對應到指定歸類，優先於自動歸類規則。',
  'route_policy.price': '單價',
  'route_policy.price_per_1k': '{{symbol}}{{price}}/1K',
  'route_policy.provider': '供應商',
  'route_policy.raw_model': '原始模型名',
  'route_policy.raw_model_placeholder': '例如 gpt-4o-2024-08-06',
  'route_policy.retry': '重試',
  'route_policy.save_failed': '儲存失敗',
  'route_policy.scope': '範圍',
  'route_policy.site_global_ref': '目前站點全域模式',
  'route_policy.title': '智能路由策略',
  'route_policy.user_weight': '我的權重',
  'route_policy.weight_deleted': '權重設定已刪除',
  'route_policy.weight_updated': '權重設定已儲存',
};

const en = {
  智能路由策略: 'Smart Routing Policy',
  'route_policy.add_override': 'Add override',
  'route_policy.channel_name': 'Channel',
  'route_policy.channels': 'channels',
  'route_policy.custom_mode_hint': 'Using a custom routing mode.',
  'route_policy.delete_failed': 'Delete failed',
  'route_policy.description':
    'Configure smart routing for model API calls under your account. Follows the system default when not customized.',
  'route_policy.disabled': 'Follow system default',
  'route_policy.disabled_hint':
    'Use the global mode and weights configured by the administrator in Smart Routing Policy.',
  'route_policy.enabled': 'Enabled',
  'route_policy.global_override': 'Global',
  'route_policy.global_weight': 'System default weight',
  'route_policy.group_key': 'Target group',
  'route_policy.group_key_placeholder': 'e.g. gpt-4o',
  'route_policy.group_models': 'Models in group',
  'route_policy.load_failed': 'Failed to load routing policy',
  'route_policy.load_failed_with_reason': 'Failed to load routing policy: {{reason}}',
  'route_policy.mode_default': 'Default (native)',
  'route_policy.mode_default_hint':
    'Your requests skip TokenFactory smart routing and use this site’s native channel selection.',
  'route_policy.mode_desc': 'Choose how model API requests select upstream channels',
  'route_policy.mode_label': 'Routing mode',
  'route_policy.mode_price': 'Price-optimized',
  'route_policy.mode_price_hint':
    'Within each group, select channels in ascending order of final unit price.',
  'route_policy.mode_reset': 'Restored to follow system default',
  'route_policy.mode_updated': 'Routing mode updated',
  'route_policy.mode_weight': 'Weighted',
  'route_policy.mode_weight_hint':
    'Sort channels by your per-group weights; falls back to system defaults when not configured.',
  'route_policy.model_groups': 'Model groups',
  'route_policy.models': 'models',
  'route_policy.my_override': 'Mine',
  'route_policy.override_added': 'Override rule added',
  'route_policy.override_deleted': 'Override rule deleted',
  'route_policy.override_required': 'Please enter raw model name and target group',
  'route_policy.overrides': 'Model group overrides',
  'route_policy.overrides_desc':
    'Map a specific raw model name to a target group, taking priority over automatic grouping.',
  'route_policy.price': 'Unit price',
  'route_policy.price_per_1k': '{{symbol}}{{price}}/1K',
  'route_policy.provider': 'Provider',
  'route_policy.raw_model': 'Raw model name',
  'route_policy.raw_model_placeholder': 'e.g. gpt-4o-2024-08-06',
  'route_policy.retry': 'Retry',
  'route_policy.save_failed': 'Save failed',
  'route_policy.scope': 'Scope',
  'route_policy.site_global_ref': 'Site global mode',
  'route_policy.title': 'Smart Routing Policy',
  'route_policy.user_weight': 'My weight',
  'route_policy.weight_deleted': 'Weight configuration deleted',
  'route_policy.weight_updated': 'Weight configuration saved',
};

const fr = {
  智能路由策略: 'Politique de routage intelligent',
  'route_policy.add_override': 'Ajouter une règle',
  'route_policy.channel_name': 'Canal',
  'route_policy.channels': 'canaux',
  'route_policy.custom_mode_hint': 'Mode de routage personnalisé actif.',
  'route_policy.delete_failed': 'Échec de la suppression',
  'route_policy.description':
    'Configurez le routage intelligent des appels API modèle pour votre compte. Suit la valeur par défaut système si non personnalisé.',
  'route_policy.disabled': 'Suivre la valeur par défaut système',
  'route_policy.disabled_hint':
    'Utilise le mode global et les poids configurés par l’administrateur dans la politique de routage intelligent.',
  'route_policy.enabled': 'Activé',
  'route_policy.global_override': 'Global',
  'route_policy.global_weight': 'Poids système par défaut',
  'route_policy.group_key': 'Groupe cible',
  'route_policy.group_key_placeholder': 'ex. gpt-4o',
  'route_policy.group_models': 'Modèles du groupe',
  'route_policy.load_failed': 'Échec du chargement de la politique de routage',
  'route_policy.load_failed_with_reason': 'Échec du chargement de la politique de routage : {{reason}}',
  'route_policy.mode_default': 'Par défaut (natif)',
  'route_policy.mode_default_hint':
    'Vos requêtes ignorent le routage intelligent TokenFactory et utilisent la sélection native du site.',
  'route_policy.mode_desc': 'Choisissez comment les requêtes API modèle sélectionnent les canaux',
  'route_policy.mode_label': 'Mode de routage',
  'route_policy.mode_price': 'Optimisé prix',
  'route_policy.mode_price_hint':
    'Dans chaque groupe, sélectionne les canaux par prix unitaire final croissant.',
  'route_policy.mode_reset': 'Rétabli sur la valeur par défaut système',
  'route_policy.mode_updated': 'Mode de routage mis à jour',
  'route_policy.mode_weight': 'Pondéré',
  'route_policy.mode_weight_hint':
    'Trie les canaux selon vos poids par groupe ; revient aux valeurs système si non configuré.',
  'route_policy.model_groups': 'Groupes de modèles',
  'route_policy.models': 'modèles',
  'route_policy.my_override': 'Moi',
  'route_policy.override_added': 'Règle de remplacement ajoutée',
  'route_policy.override_deleted': 'Règle de remplacement supprimée',
  'route_policy.override_required': 'Veuillez saisir le nom de modèle brut et le groupe cible',
  'route_policy.overrides': 'Remplacements de groupe de modèles',
  'route_policy.overrides_desc':
    'Associe un nom de modèle brut à un groupe cible, prioritaire sur le regroupement automatique.',
  'route_policy.price': 'Prix unitaire',
  'route_policy.price_per_1k': '{{symbol}}{{price}}/1K',
  'route_policy.provider': 'Fournisseur',
  'route_policy.raw_model': 'Nom de modèle brut',
  'route_policy.raw_model_placeholder': 'ex. gpt-4o-2024-08-06',
  'route_policy.retry': 'Réessayer',
  'route_policy.save_failed': 'Échec de l’enregistrement',
  'route_policy.scope': 'Portée',
  'route_policy.site_global_ref': 'Mode global du site',
  'route_policy.title': 'Politique de routage intelligent',
  'route_policy.user_weight': 'Mon poids',
  'route_policy.weight_deleted': 'Configuration de poids supprimée',
  'route_policy.weight_updated': 'Configuration de poids enregistrée',
};

const ja = {
  智能路由策略: 'スマートルーティングポリシー',
  'route_policy.add_override': '上書きルールを追加',
  'route_policy.channel_name': 'チャネル',
  'route_policy.channels': 'チャネル',
  'route_policy.custom_mode_hint': 'カスタムルーティングモードを使用中です。',
  'route_policy.delete_failed': '削除に失敗しました',
  'route_policy.description':
    'アカウントのモデル API 呼び出し向けスマートルーティングを設定します。未設定時はシステム既定に従います。',
  'route_policy.disabled': 'システム既定に従う',
  'route_policy.disabled_hint':
    '管理者がスマートルーティングポリシーで設定したグローバルモードと重みを使用します。',
  'route_policy.enabled': '有効',
  'route_policy.global_override': 'グローバル',
  'route_policy.global_weight': 'システム既定の重み',
  'route_policy.group_key': '対象グループ',
  'route_policy.group_key_placeholder': '例: gpt-4o',
  'route_policy.group_models': 'グループ内モデル',
  'route_policy.load_failed': 'ルーティングポリシーの読み込みに失敗しました',
  'route_policy.load_failed_with_reason': 'ルーティングポリシーの読み込みに失敗しました：{{reason}}',
  'route_policy.mode_default': '既定（ネイティブ）',
  'route_policy.mode_default_hint':
    'TokenFactory のスマートルーティングを使わず、このサイトのネイティブ選路を使用します。',
  'route_policy.mode_desc': 'モデル API リクエストのチャネル選択方法を選びます',
  'route_policy.mode_label': 'ルーティングモード',
  'route_policy.mode_price': '価格優先',
  'route_policy.mode_price_hint': '各グループ内で最終単価の昇順にチャネルを選択します。',
  'route_policy.mode_reset': 'システム既定に戻しました',
  'route_policy.mode_updated': 'ルーティングモードを更新しました',
  'route_policy.mode_weight': '重み付け',
  'route_policy.mode_weight_hint':
    'グループごとの重みでチャネルを並べ替えます。未設定時はシステム既定にフォールバックします。',
  'route_policy.model_groups': 'モデルグループ',
  'route_policy.models': 'モデル',
  'route_policy.my_override': '自分',
  'route_policy.override_added': '上書きルールを追加しました',
  'route_policy.override_deleted': '上書きルールを削除しました',
  'route_policy.override_required': '生のモデル名と対象グループを入力してください',
  'route_policy.overrides': 'モデルグループ上書き',
  'route_policy.overrides_desc':
    '特定の生モデル名を対象グループにマッピングし、自動グループ化より優先します。',
  'route_policy.price': '単価',
  'route_policy.price_per_1k': '{{symbol}}{{price}}/1K',
  'route_policy.provider': 'プロバイダー',
  'route_policy.raw_model': '生のモデル名',
  'route_policy.raw_model_placeholder': '例: gpt-4o-2024-08-06',
  'route_policy.retry': '再試行',
  'route_policy.save_failed': '保存に失敗しました',
  'route_policy.scope': '範囲',
  'route_policy.site_global_ref': 'サイトのグローバルモード',
  'route_policy.title': 'スマートルーティングポリシー',
  'route_policy.user_weight': '自分の重み',
  'route_policy.weight_deleted': '重み設定を削除しました',
  'route_policy.weight_updated': '重み設定を保存しました',
};

const ru = {
  智能路由策略: 'Политика умной маршрутизации',
  'route_policy.add_override': 'Добавить правило',
  'route_policy.channel_name': 'Канал',
  'route_policy.channels': 'каналов',
  'route_policy.custom_mode_hint': 'Используется пользовательский режим маршрутизации.',
  'route_policy.delete_failed': 'Не удалось удалить',
  'route_policy.description':
    'Настройте умную маршрутизацию вызовов API моделей для вашего аккаунта. Без настройки используется системное значение по умолчанию.',
  'route_policy.disabled': 'Следовать системному умолчанию',
  'route_policy.disabled_hint':
    'Используются глобальный режим и веса, настроенные администратором в политике умной маршрутизации.',
  'route_policy.enabled': 'Включено',
  'route_policy.global_override': 'Глобально',
  'route_policy.global_weight': 'Системный вес по умолчанию',
  'route_policy.group_key': 'Целевая группа',
  'route_policy.group_key_placeholder': 'напр. gpt-4o',
  'route_policy.group_models': 'Модели в группе',
  'route_policy.load_failed': 'Не удалось загрузить политику маршрутизации',
  'route_policy.load_failed_with_reason': 'Не удалось загрузить политику маршрутизации: {{reason}}',
  'route_policy.mode_default': 'По умолчанию (нативный)',
  'route_policy.mode_default_hint':
    'Ваши запросы не используют умную маршрутизацию TokenFactory и выбирают канал нативной логикой сайта.',
  'route_policy.mode_desc': 'Выберите способ выбора каналов для запросов API моделей',
  'route_policy.mode_label': 'Режим маршрутизации',
  'route_policy.mode_price': 'По цене',
  'route_policy.mode_price_hint':
    'В каждой группе каналы выбираются по возрастанию итоговой цены за единицу.',
  'route_policy.mode_reset': 'Восстановлено следование системному умолчанию',
  'route_policy.mode_updated': 'Режим маршрутизации обновлён',
  'route_policy.mode_weight': 'По весам',
  'route_policy.mode_weight_hint':
    'Сортировка каналов по вашим весам для групп; при отсутствии настройки — системные значения.',
  'route_policy.model_groups': 'Группы моделей',
  'route_policy.models': 'моделей',
  'route_policy.my_override': 'Моё',
  'route_policy.override_added': 'Правило переопределения добавлено',
  'route_policy.override_deleted': 'Правило переопределения удалено',
  'route_policy.override_required': 'Укажите исходное имя модели и целевую группу',
  'route_policy.overrides': 'Переопределения групп моделей',
  'route_policy.overrides_desc':
    'Сопоставляет исходное имя модели с целевой группой, приоритетнее автоматической группировки.',
  'route_policy.price': 'Цена за единицу',
  'route_policy.price_per_1k': '{{symbol}}{{price}}/1K',
  'route_policy.provider': 'Поставщик',
  'route_policy.raw_model': 'Исходное имя модели',
  'route_policy.raw_model_placeholder': 'напр. gpt-4o-2024-08-06',
  'route_policy.retry': 'Повторить',
  'route_policy.save_failed': 'Не удалось сохранить',
  'route_policy.scope': 'Область',
  'route_policy.site_global_ref': 'Глобальный режим сайта',
  'route_policy.title': 'Политика умной маршрутизации',
  'route_policy.user_weight': 'Мой вес',
  'route_policy.weight_deleted': 'Настройка веса удалена',
  'route_policy.weight_updated': 'Настройка веса сохранена',
};

const vi = {
  智能路由策略: 'Chính sách định tuyến thông minh',
  'route_policy.add_override': 'Thêm quy tắc ghi đè',
  'route_policy.channel_name': 'Kênh',
  'route_policy.channels': 'kênh',
  'route_policy.custom_mode_hint': 'Đang dùng chế độ định tuyến tùy chỉnh.',
  'route_policy.delete_failed': 'Xóa thất bại',
  'route_policy.description':
    'Cấu hình định tuyến thông minh cho lời gọi API mô hình trong tài khoản của bạn. Không cấu hình thì theo mặc định hệ thống.',
  'route_policy.disabled': 'Theo mặc định hệ thống',
  'route_policy.disabled_hint':
    'Dùng chế độ toàn cục và trọng số do quản trị viên cấu hình trong chính sách định tuyến thông minh.',
  'route_policy.enabled': 'Bật',
  'route_policy.global_override': 'Toàn cục',
  'route_policy.global_weight': 'Trọng số mặc định hệ thống',
  'route_policy.group_key': 'Nhóm đích',
  'route_policy.group_key_placeholder': 'vd. gpt-4o',
  'route_policy.group_models': 'Mô hình trong nhóm',
  'route_policy.load_failed': 'Không tải được chính sách định tuyến',
  'route_policy.load_failed_with_reason': 'Không tải được chính sách định tuyến: {{reason}}',
  'route_policy.mode_default': 'Mặc định (gốc)',
  'route_policy.mode_default_hint':
    'Yêu cầu của bạn không dùng định tuyến thông minh TokenFactory mà dùng logic chọn kênh gốc của trang.',
  'route_policy.mode_desc': 'Chọn cách yêu cầu API mô hình chọn kênh upstream',
  'route_policy.mode_label': 'Chế độ định tuyến',
  'route_policy.mode_price': 'Ưu tiên giá',
  'route_policy.mode_price_hint':
    'Trong mỗi nhóm, chọn kênh theo thứ tự giá đơn vị cuối cùng tăng dần.',
  'route_policy.mode_reset': 'Đã khôi phục theo mặc định hệ thống',
  'route_policy.mode_updated': 'Đã cập nhật chế độ định tuyến',
  'route_policy.mode_weight': 'Theo trọng số',
  'route_policy.mode_weight_hint':
    'Sắp xếp kênh theo trọng số từng nhóm của bạn; chưa cấu hình thì dùng mặc định hệ thống.',
  'route_policy.model_groups': 'Nhóm mô hình',
  'route_policy.models': 'mô hình',
  'route_policy.my_override': 'Của tôi',
  'route_policy.override_added': 'Đã thêm quy tắc ghi đè',
  'route_policy.override_deleted': 'Đã xóa quy tắc ghi đè',
  'route_policy.override_required': 'Vui lòng nhập tên mô hình gốc và nhóm đích',
  'route_policy.overrides': 'Ghi đè nhóm mô hình',
  'route_policy.overrides_desc':
    'Ánh xạ tên mô hình gốc cụ thể vào nhóm đích, ưu tiên hơn nhóm tự động.',
  'route_policy.price': 'Đơn giá',
  'route_policy.price_per_1k': '{{symbol}}{{price}}/1K',
  'route_policy.provider': 'Nhà cung cấp',
  'route_policy.raw_model': 'Tên mô hình gốc',
  'route_policy.raw_model_placeholder': 'vd. gpt-4o-2024-08-06',
  'route_policy.retry': 'Thử lại',
  'route_policy.save_failed': 'Lưu thất bại',
  'route_policy.scope': 'Phạm vi',
  'route_policy.site_global_ref': 'Chế độ toàn cục của trang',
  'route_policy.title': 'Chính sách định tuyến thông minh',
  'route_policy.user_weight': 'Trọng số của tôi',
  'route_policy.weight_deleted': 'Đã xóa cấu hình trọng số',
  'route_policy.weight_updated': 'Đã lưu cấu hình trọng số',
};

const localeMap = {
  'zh-CN.json': zhCN,
  'zh-TW.json': zhTW,
  'en.json': en,
  'fr.json': fr,
  'ru.json': ru,
  'ja.json': ja,
  'vi.json': vi,
  'id.json': en,
  'ms.json': en,
  'th.json': en,
  'sw.json': en,
};

for (const [file, entries] of Object.entries(localeMap)) {
  const filePath = path.join(localesDir, file);
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  const translation = data.translation ?? data;
  Object.assign(translation, entries);
  const sorted = Object.fromEntries(
    Object.entries(translation).sort(([a], [b]) => a.localeCompare(b, 'zh-CN')),
  );
  if (data.translation) {
    data.translation = sorted;
  } else {
    Object.keys(data).forEach((k) => delete data[k]);
    Object.assign(data, sorted);
  }
  fs.writeFileSync(filePath, `${JSON.stringify(data, null, 4)}\n`, 'utf8');
  console.log(`Updated ${file}`);
}
