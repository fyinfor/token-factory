package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	userImportSheetName   = "用户导入"
	userImportMaxFileSize = 10 << 20
)

var userImportHeaders = []string{"用户名", "密码", "显示名称", "邮箱", "手机号", "分组", "标签", "是否为学员", "是否为代理", "备注"}

type userImportTemplateLocale struct {
	Lang                 string
	SheetName            string
	InfoSheetName        string
	OptionsSheetName     string
	Headers              []string
	FailureReason        string
	ExampleUser          string
	ExampleRemark        string
	Notes                [][]string
	Yes                  string
	No                   string
	GroupInputTitle      string
	GroupInputMessage    string
	BoolInputTitle       string
	BoolInputMessage     string
	PasswordInputTitle   string
	PasswordInputMessage string
	PasswordWarningTitle string
	PasswordWarningMsg   string
}

func userImportLocales() map[string]userImportTemplateLocale {
	return map[string]userImportTemplateLocale{
		"zh-CN": newUserImportLocale("zh-CN", "用户导入", "填写说明", "选项",
			[]string{"用户名", "密码", "显示名称", "邮箱", "手机号", "分组", "标签", "是否为学员", "是否为代理", "备注"},
			"失败原因", "示例用户", "这是示例行，导入时会自动跳过；请从第 3 行开始填写真实用户",
			[]string{
				"字段|说明",
				"用户名|必填，最长 20 个字符，不能与已存在或已注销用户重复",
				"密码|必填，8-20 个字符",
				"显示名称|选填，留空则使用用户名",
				"邮箱|选填，填写时必须格式正确，不能与未注销用户重复",
				"手机号|选填，仅支持 11 位中国大陆手机号，不能与未注销用户重复",
				"分组|选填，可从下拉列表选择；留空默认为 default",
				"标签|选填，多个标签可用逗号、分号或竖线分隔",
				"是否为学员|选填，下拉选择 是/否。选择“是”导入后即为已通过学员",
				"是否为代理|选填，下拉选择 是/否。选择“是”导入后即开通代理身份",
				"备注|选填，仅管理员可见，最长 255 个字符",
			}, "是", "否", "请选择分组", "可从下拉列表选择系统分组；留空默认 default", "请选择", "只能选择“是”或“否”，留空按“否”处理", "密码要求", "必填，长度 8-20 个字符", "密码格式提示", "密码长度应为 8-20 个字符"),
		"zh-TW": newUserImportLocale("zh-TW", "使用者匯入", "填寫說明", "選項",
			[]string{"使用者名稱", "密碼", "顯示名稱", "電子郵件", "手機號", "分組", "標籤", "是否為學員", "是否為代理", "備註"},
			"失敗原因", "範例使用者", "這是範例列，匯入時會自動跳過；請從第 3 列開始填寫真實使用者",
			[]string{
				"欄位|說明",
				"使用者名稱|必填，最長 20 個字元，不能與已存在或已註銷使用者重複",
				"密碼|必填，8-20 個字元",
				"顯示名稱|選填，留空則使用使用者名稱",
				"電子郵件|選填，填寫時格式必須正確，不能與未註銷使用者重複",
				"手機號|選填，僅支援 11 位中國大陸手機號，不能與未註銷使用者重複",
				"分組|選填，可從下拉列表選擇；留空預設為 default",
				"標籤|選填，多個標籤可用逗號、分號或直線分隔",
				"是否為學員|選填，下拉選擇 是/否。選擇「是」匯入後即為已通過學員",
				"是否為代理|選填，下拉選擇 是/否。選擇「是」匯入後即開通代理身分",
				"備註|選填，僅管理員可見，最長 255 個字元",
			}, "是", "否", "請選擇分組", "可從下拉列表選擇系統分組；留空預設 default", "請選擇", "只能選擇「是」或「否」，留空按「否」處理", "密碼要求", "必填，長度 8-20 個字元", "密碼格式提示", "密碼長度應為 8-20 個字元"),
		"en": newUserImportLocale("en", "User Import", "Instructions", "Options",
			[]string{"Username", "Password", "Display Name", "Email", "Phone", "Group", "Tags", "Is Student", "Is Agent", "Remark"},
			"Failure Reason", "Example User", "This is an example row and will be skipped during import; start real users from row 3.",
			[]string{
				"Field|Description",
				"Username|Required, up to 20 characters, must not duplicate an existing or deleted user",
				"Password|Required, 8-20 characters",
				"Display Name|Optional; username is used when blank",
				"Email|Optional; must be valid and must not duplicate an active user",
				"Phone|Optional; supports 11-digit mainland China phone numbers only and must not duplicate an active user",
				"Group|Optional; choose from the dropdown. Blank defaults to default",
				"Tags|Optional; separate multiple tags with comma, semicolon, or vertical bar",
				"Is Student|Optional; choose Yes/No. Yes imports the user as an approved student",
				"Is Agent|Optional; choose Yes/No. Yes enables agent identity",
				"Remark|Optional; visible to admins only, up to 255 characters",
			}, "Yes", "No", "Select group", "Choose a system group from the dropdown; blank defaults to default", "Select", "Only Yes or No is allowed; blank is treated as No", "Password requirement", "Required, 8-20 characters", "Password format", "Password length should be 8-20 characters"),
		"fr": newUserImportLocale("fr", "Import utilisateurs", "Instructions", "Options",
			[]string{"Nom d'utilisateur", "Mot de passe", "Nom affiche", "E-mail", "Telephone", "Groupe", "Etiquettes", "Est etudiant", "Est agent", "Remarque"},
			"Raison de l'echec", "Utilisateur exemple", "Cette ligne d'exemple sera ignoree lors de l'import ; commencez les vrais utilisateurs a la ligne 3.",
			[]string{
				"Champ|Description",
				"Nom d'utilisateur|Obligatoire, 20 caracteres maximum, ne doit pas dupliquer un utilisateur existant ou supprime",
				"Mot de passe|Obligatoire, 8-20 caracteres",
				"Nom affiche|Facultatif ; le nom d'utilisateur est utilise si vide",
				"E-mail|Facultatif ; format valide et non duplique avec un utilisateur actif",
				"Telephone|Facultatif ; uniquement numeros mobiles chinois continentaux a 11 chiffres, non dupliques",
				"Groupe|Facultatif ; choisissez dans la liste. Vide vaut default",
				"Etiquettes|Facultatif ; separez plusieurs etiquettes par virgule, point-virgule ou barre verticale",
				"Est etudiant|Facultatif ; choisissez Oui/Non. Oui importe l'utilisateur comme etudiant approuve",
				"Est agent|Facultatif ; choisissez Oui/Non. Oui active l'identite agent",
				"Remarque|Facultatif ; visible uniquement par les administrateurs, 255 caracteres maximum",
			}, "Oui", "Non", "Selectionner un groupe", "Choisissez un groupe systeme dans la liste ; vide vaut default", "Selectionner", "Seuls Oui ou Non sont autorises ; vide vaut Non", "Exigence du mot de passe", "Obligatoire, 8-20 caracteres", "Format du mot de passe", "Le mot de passe doit contenir 8-20 caracteres"),
		"ru": newUserImportLocale("ru", "Импорт пользователей", "Инструкции", "Опции",
			[]string{"Имя пользователя", "Пароль", "Отображаемое имя", "Эл. почта", "Телефон", "Группа", "Теги", "Студент", "Агент", "Примечание"},
			"Причина ошибки", "Пример пользователя", "Эта строка является примером и будет пропущена при импорте; реальные пользователи начинаются со строки 3.",
			[]string{
				"Поле|Описание",
				"Имя пользователя|Обязательно, до 20 символов, не должно совпадать с существующим или удаленным пользователем",
				"Пароль|Обязательно, 8-20 символов",
				"Отображаемое имя|Необязательно; если пусто, используется имя пользователя",
				"Эл. почта|Необязательно; корректный формат, не должна совпадать с активным пользователем",
				"Телефон|Необязательно; только 11-значные номера материкового Китая, не должны совпадать с активным пользователем",
				"Группа|Необязательно; выберите из списка. Пусто означает default",
				"Теги|Необязательно; несколько тегов разделяйте запятой, точкой с запятой или вертикальной чертой",
				"Студент|Необязательно; выберите Да/Нет. Да импортирует пользователя как одобренного студента",
				"Агент|Необязательно; выберите Да/Нет. Да включает статус агента",
				"Примечание|Необязательно; видно только администраторам, до 255 символов",
			}, "Да", "Нет", "Выберите группу", "Выберите системную группу из списка; пусто означает default", "Выберите", "Допустимо только Да или Нет; пусто считается Нет", "Требование к паролю", "Обязательно, 8-20 символов", "Формат пароля", "Длина пароля должна быть 8-20 символов"),
		"ja": newUserImportLocale("ja", "ユーザーインポート", "入力説明", "オプション",
			[]string{"ユーザー名", "パスワード", "表示名", "メール", "電話番号", "グループ", "タグ", "学生か", "代理か", "備考"},
			"失敗理由", "サンプルユーザー", "これはサンプル行です。インポート時に自動的にスキップされます。実データは 3 行目から入力してください。",
			[]string{
				"項目|説明",
				"ユーザー名|必須、最大 20 文字。既存または削除済みユーザーと重複不可",
				"パスワード|必須、8-20 文字",
				"表示名|任意。空の場合はユーザー名を使用",
				"メール|任意。正しい形式で、アクティブユーザーと重複不可",
				"電話番号|任意。中国本土の 11 桁携帯番号のみ対応、アクティブユーザーと重複不可",
				"グループ|任意。ドロップダウンから選択。空の場合は default",
				"タグ|任意。複数タグはカンマ、セミコロン、縦線で区切ります",
				"学生か|任意。はい/いいえを選択。はいの場合、承認済み学生としてインポート",
				"代理か|任意。はい/いいえを選択。はいの場合、代理権限を有効化",
				"備考|任意。管理者のみ閲覧可能、最大 255 文字",
			}, "はい", "いいえ", "グループを選択", "ドロップダウンからシステムグループを選択。空の場合は default", "選択", "はい/いいえのみ選択できます。空の場合は いいえ として扱います", "パスワード要件", "必須、8-20 文字", "パスワード形式", "パスワードは 8-20 文字にしてください"),
		"vi": newUserImportLocale("vi", "Nhap nguoi dung", "Huong dan", "Tuy chon",
			[]string{"Ten dang nhap", "Mat khau", "Ten hien thi", "Email", "So dien thoai", "Nhom", "Nhan", "La hoc vien", "La dai ly", "Ghi chu"},
			"Ly do that bai", "Nguoi dung mau", "Day la dong vi du va se duoc bo qua khi nhap; hay bat dau nguoi dung that tu dong 3.",
			[]string{
				"Truong|Mo ta",
				"Ten dang nhap|Bat buoc, toi da 20 ky tu, khong duoc trung voi nguoi dung ton tai hoac da xoa",
				"Mat khau|Bat buoc, 8-20 ky tu",
				"Ten hien thi|Tuy chon; de trong se dung ten dang nhap",
				"Email|Tuy chon; dung dinh dang va khong trung voi nguoi dung dang hoat dong",
				"So dien thoai|Tuy chon; chi ho tro so di dong Trung Quoc dai luc 11 chu so, khong trung voi nguoi dung dang hoat dong",
				"Nhom|Tuy chon; chon tu danh sach. De trong mac dinh la default",
				"Nhan|Tuy chon; nhieu nhan phan tach bang dau phay, cham phay hoac gach doc",
				"La hoc vien|Tuy chon; chon Co/Khong. Co se nhap nguoi dung thanh hoc vien da duyet",
				"La dai ly|Tuy chon; chon Co/Khong. Co se bat danh tinh dai ly",
				"Ghi chu|Tuy chon; chi quan tri vien thay, toi da 255 ky tu",
			}, "Co", "Khong", "Chon nhom", "Chon nhom he thong tu danh sach; de trong mac dinh la default", "Chon", "Chi duoc chon Co hoac Khong; de trong xem la Khong", "Yeu cau mat khau", "Bat buoc, 8-20 ky tu", "Dinh dang mat khau", "Mat khau phai dai 8-20 ky tu"),
	}
}

func newUserImportLocale(lang, sheetName, infoSheetName, optionsSheetName string, headers []string, failureReason, exampleUser, exampleRemark string, notesRaw []string, yes, no, groupInputTitle, groupInputMessage, boolInputTitle, boolInputMessage, passwordInputTitle, passwordInputMessage, passwordWarningTitle, passwordWarningMsg string) userImportTemplateLocale {
	notes := make([][]string, 0, len(notesRaw))
	for _, raw := range notesRaw {
		parts := strings.SplitN(raw, "|", 2)
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		notes = append(notes, parts)
	}
	return userImportTemplateLocale{
		Lang:                 lang,
		SheetName:            sheetName,
		InfoSheetName:        infoSheetName,
		OptionsSheetName:     optionsSheetName,
		Headers:              headers,
		FailureReason:        failureReason,
		ExampleUser:          exampleUser,
		ExampleRemark:        exampleRemark,
		Notes:                notes,
		Yes:                  yes,
		No:                   no,
		GroupInputTitle:      groupInputTitle,
		GroupInputMessage:    groupInputMessage,
		BoolInputTitle:       boolInputTitle,
		BoolInputMessage:     boolInputMessage,
		PasswordInputTitle:   passwordInputTitle,
		PasswordInputMessage: passwordInputMessage,
		PasswordWarningTitle: passwordWarningTitle,
		PasswordWarningMsg:   passwordWarningMsg,
	}
}

func getUserImportTemplateLocale(c *gin.Context) userImportTemplateLocale {
	lang := strings.TrimSpace(c.Query("lang"))
	if lang == "" {
		lang = strings.TrimSpace(c.GetHeader("Accept-Language"))
	}
	lang = normalizeUserImportLang(lang)
	locales := userImportLocales()
	if locale, ok := locales[lang]; ok {
		return locale
	}
	return locales["en"]
}

func normalizeUserImportLang(lang string) string {
	lang = strings.TrimSpace(strings.ReplaceAll(lang, "_", "-"))
	if lang == "" {
		return "zh-CN"
	}
	first := strings.Split(lang, ",")[0]
	first = strings.TrimSpace(strings.Split(first, ";")[0])
	lower := strings.ToLower(first)
	switch {
	case lower == "zh" || lower == "zh-cn" || lower == "zh-sg" || strings.HasPrefix(lower, "zh-hans"):
		return "zh-CN"
	case lower == "zh-tw" || lower == "zh-hk" || lower == "zh-mo" || strings.HasPrefix(lower, "zh-hant"):
		return "zh-TW"
	case strings.HasPrefix(lower, "fr"):
		return "fr"
	case strings.HasPrefix(lower, "ru"):
		return "ru"
	case strings.HasPrefix(lower, "ja"):
		return "ja"
	case strings.HasPrefix(lower, "vi"):
		return "vi"
	case strings.HasPrefix(lower, "en"):
		return "en"
	default:
		return lower
	}
}

type userImportFailure struct {
	Row         int    `json:"row"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Group       string `json:"group"`
	Tags        string `json:"tags"`
	IsStudent   string `json:"is_student"`
	Reward      string `json:"student_reward_amount"`
	IsAgent     string `json:"is_agent"`
	Remark      string `json:"remark"`
	Reason      string `json:"reason"`
}

type userImportFailureFile struct {
	bytes     []byte
	createdAt time.Time
}

var (
	userImportFailureFilesMu sync.Mutex
	userImportFailureFiles   = map[string]userImportFailureFile{}
)

func DownloadUserImportTemplate(c *gin.Context) {
	file, err := buildUserImportWorkbook(nil, getUserImportTemplateLocale(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	writeUserImportWorkbook(c, file, "user_import_template.xlsx")
}

func ImportUsers(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请选择要导入的 Excel 文件")
		return
	}
	if fileHeader.Size > userImportMaxFileSize {
		common.ApiErrorMsg(c, "Excel 文件不能超过 10MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".xlsx" && ext != ".xlsm" {
		common.ApiErrorMsg(c, "仅支持 .xlsx 或 .xlsm 文件")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer file.Close()

	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		common.ApiErrorMsg(c, "Excel 文件解析失败，请确认文件格式正确")
		return
	}
	defer xlsx.Close()

	sheet := userImportSheetName
	sheetIndex, err := xlsx.GetSheetIndex(sheet)
	if err != nil || sheetIndex == -1 {
		sheets := xlsx.GetSheetList()
		if len(sheets) == 0 {
			common.ApiErrorMsg(c, "Excel 文件没有工作表")
			return
		}
		sheet = sheets[0]
	}

	rows, err := xlsx.GetRows(sheet)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(rows) < 2 {
		common.ApiErrorMsg(c, "Excel 文件没有可导入的数据")
		return
	}

	headerMap := buildUserImportHeaderMap(rows[0])
	if _, ok := headerMap["username"]; !ok {
		common.ApiErrorMsg(c, "模板缺少“用户名”列")
		return
	}
	if _, ok := headerMap["password"]; !ok {
		common.ApiErrorMsg(c, "模板缺少“密码”列")
		return
	}

	created := 0
	total := 0
	failures := make([]userImportFailure, 0)
	seenUsernames := map[string]int{}
	seenEmails := map[string]int{}
	seenPhones := map[string]int{}
	createdTags := make([]string, 0)

	for idx := 1; idx < len(rows); idx++ {
		row := rows[idx]
		if isUserImportRowEmpty(row) {
			continue
		}
		if isUserImportExampleRow(row, headerMap) {
			continue
		}
		total++
		item := parseUserImportRow(row, headerMap, idx+1)
		if reason := validateUserImportItem(&item, seenUsernames, seenEmails, seenPhones); reason != "" {
			item.Reason = reason
			failures = append(failures, item)
			continue
		}

		cleanUser := model.User{
			Username:                   item.Username,
			Password:                   item.Password,
			DisplayName:                item.DisplayName,
			Role:                       common.RoleCommonUser,
			Status:                     common.UserStatusEnabled,
			CreatedBy:                  common.UserCreatedByAdmin,
			Phone:                      item.Phone,
			Email:                      item.Email,
			Group:                      item.Group,
			Tags:                       item.Tags,
			IsDistributor:              common.DistributorFlagNo,
			Remark:                     item.Remark,
			AdminInitialSetupCompleted: false,
		}
		if yes, _ := parseUserImportBool(item.IsAgent); yes {
			cleanUser.IsDistributor = common.DistributorFlagYes
		}
		if yes, _ := parseUserImportBool(item.IsStudent); yes {
			now := time.Now()
			cleanUser.IsStudent = 1
			cleanUser.StudentStatus = common.StudentStatusApproved
			cleanUser.StudentApplied = &now
			cleanUser.StudentApprovedAt = &now
			cleanUser.StudentApprovedBy = c.GetInt("id")
		}
		if err := cleanUser.Insert(0); err != nil {
			item.Reason = err.Error()
			failures = append(failures, item)
			continue
		}
		if rewardQuota, _ := parseUserImportRewardQuota(item.Reward); rewardQuota > 0 {
			if err := model.IncreaseUserQuota(cleanUser.Id, rewardQuota, true); err != nil {
				common.SysLog(fmt.Sprintf("failed to grant imported student reward: user_id=%d quota=%d error=%v", cleanUser.Id, rewardQuota, err))
			} else {
				model.RecordLog(cleanUser.Id, model.LogTypeManage, fmt.Sprintf("批量导入学员身份，赠送 %s", logger.LogQuota(rewardQuota)))
			}
		}
		created++
		createdTags = append(createdTags, model.GetUserTagsList(item.Tags)...)
	}

	if len(createdTags) > 0 {
		if err := model.UpsertUserTags(createdTags); err != nil {
			common.SysLog("failed to upsert imported user tags: " + err.Error())
		}
	}

	failureID := ""
	if len(failures) > 0 {
		failureWorkbook, err := buildUserImportWorkbook(failures, getUserImportTemplateLocale(c))
		if err != nil {
			common.ApiError(c, err)
			return
		}
		failureID = common.GetRandomString(24)
		storeUserImportFailureFile(failureID, failureWorkbook)
	}

	common.ApiSuccess(c, gin.H{
		"total":      total,
		"created":    created,
		"failed":     len(failures),
		"failures":   failures,
		"failure_id": failureID,
	})
}

func DownloadUserImportFailures(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		common.ApiErrorMsg(c, "失败列表不存在或已过期")
		return
	}
	data, ok := getUserImportFailureFile(id)
	if !ok {
		common.ApiErrorMsg(c, "失败列表不存在或已过期，请重新导入")
		return
	}
	writeUserImportWorkbook(c, data, "user_import_failures.xlsx")
}

func buildUserImportHeaderMap(row []string) map[string]int {
	headerMap := make(map[string]int, len(row))
	for i, cell := range row {
		key := normalizeUserImportHeader(cell)
		if key != "" {
			headerMap[key] = i
		}
	}
	return headerMap
}

func normalizeUserImportHeader(header string) string {
	normalized := strings.ToLower(strings.TrimSpace(header))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "'", "")
	normalized = strings.ReplaceAll(normalized, "’", "")
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, "(", "")
	normalized = strings.ReplaceAll(normalized, ")", "")
	normalized = strings.ReplaceAll(normalized, "（", "")
	normalized = strings.ReplaceAll(normalized, "）", "")
	if strings.Contains(normalized, "学员奖励金额") ||
		strings.Contains(normalized, "學員獎勵金額") ||
		strings.Contains(normalized, "studentreward") ||
		strings.Contains(normalized, "rewardamount") ||
		strings.Contains(normalized, "montantrecompenseetudiant") ||
		strings.Contains(normalized, "вознаграждениестудента") ||
		strings.Contains(normalized, "学生報酬額") ||
		strings.Contains(normalized, "thuonghocvien") {
		return "student_reward_amount"
	}
	switch normalized {
	case "用户名", "用户名称", "使用者名稱", "username", "nomdutilisateur", "имяпользователя", "ユーザー名", "tendangnhap":
		return "username"
	case "密码", "密碼", "password", "motdepasse", "пароль", "パスワード", "matkhau":
		return "password"
	case "显示名称", "显示名", "顯示名稱", "displayname", "nomaffiche", "отображаемоеимя", "表示名", "tenhienthi":
		return "display_name"
	case "邮箱", "电子邮箱", "電子郵件", "email", "элпочта", "メール":
		return "email"
	case "手机号", "手机", "电话", "手機號", "phone", "telephone", "телефон", "電話番号", "sodienthoai":
		return "phone"
	case "分组", "分組", "group", "groupe", "группа", "グループ", "nhom":
		return "group"
	case "标签", "標籤", "tags", "tag", "etiquettes", "теги", "タグ", "nhan":
		return "tags"
	case "是否为学员", "学员", "是否為學員", "isstudent", "student", "estetudiant", "студент", "学生か", "lahocvien":
		return "is_student"
	case "学员奖励金额", "學員獎勵金額", "studentrewardamount", "studentreward", "rewardamount", "reward", "montantdelarecompenseetudiant", "récompenseétudiant", "вознаграждениестудента", "学生報酬額", "thuonghocvien":
		return "student_reward_amount"
	case "是否为代理", "代理", "是否为分销商", "分销商", "是否為代理", "isagent", "isdistributor", "distributor", "estagent", "агент", "代理か", "ladaily":
		return "is_agent"
	case "备注", "備註", "remark", "note", "remarque", "примечание", "備考", "ghichu":
		return "remark"
	case "失败原因", "失敗原因", "原因", "reason", "raisondelechec", "причинаошибки", "失敗理由", "lydothatbai":
		return "reason"
	default:
		return ""
	}
}

func parseUserImportRow(row []string, headerMap map[string]int, rowNumber int) userImportFailure {
	get := func(key string) string {
		idx, ok := headerMap[key]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	group := get("group")
	if group == "" {
		group = "default"
	}
	tags := normalizeUserImportTags(get("tags"))
	displayName := get("display_name")
	username := get("username")
	if displayName == "" {
		displayName = username
	}
	return userImportFailure{
		Row:         rowNumber,
		Username:    username,
		Password:    get("password"),
		DisplayName: displayName,
		Email:       get("email"),
		Phone:       get("phone"),
		Group:       group,
		Tags:        tags,
		IsStudent:   normalizeUserImportBoolText(get("is_student")),
		Reward:      strings.TrimSpace(get("student_reward_amount")),
		IsAgent:     normalizeUserImportBoolText(get("is_agent")),
		Remark:      get("remark"),
	}
}

func isUserImportRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func isUserImportExampleRow(row []string, headerMap map[string]int) bool {
	get := func(key string) string {
		idx, ok := headerMap[key]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	return strings.EqualFold(get("username"), "example_user") && get("password") == "password123"
}

func validateUserImportItem(item *userImportFailure, seenUsernames map[string]int, seenEmails map[string]int, seenPhones map[string]int) string {
	item.Username = strings.TrimSpace(item.Username)
	item.Password = strings.TrimSpace(item.Password)
	item.DisplayName = strings.TrimSpace(item.DisplayName)
	item.Email = strings.TrimSpace(item.Email)
	item.Phone = common.NormalizePhone(item.Phone)
	item.Group = strings.TrimSpace(item.Group)
	item.Tags = normalizeUserImportTags(item.Tags)
	item.IsStudent = normalizeUserImportBoolText(item.IsStudent)
	item.Reward = strings.TrimSpace(item.Reward)
	item.IsAgent = normalizeUserImportBoolText(item.IsAgent)
	item.Remark = strings.TrimSpace(item.Remark)

	if item.Username == "" {
		return "用户名不能为空"
	}
	if item.Password == "" {
		return "密码不能为空"
	}
	if item.DisplayName == "" {
		item.DisplayName = item.Username
	}
	if item.Group == "" {
		item.Group = "default"
	}
	if prev, ok := seenUsernames[item.Username]; ok {
		return fmt.Sprintf("用户名与第 %d 行重复", prev)
	}
	if item.Email != "" {
		key := strings.ToLower(item.Email)
		if prev, ok := seenEmails[key]; ok {
			return fmt.Sprintf("邮箱与第 %d 行重复", prev)
		}
	}
	if item.Phone != "" {
		if prev, ok := seenPhones[item.Phone]; ok {
			return fmt.Sprintf("手机号与第 %d 行重复", prev)
		}
	}

	user := model.User{
		Username:    item.Username,
		Password:    item.Password,
		DisplayName: item.DisplayName,
		Email:       item.Email,
		Phone:       item.Phone,
		Group:       item.Group,
		Tags:        item.Tags,
		Remark:      item.Remark,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	if err := common.Validate.Struct(&user); err != nil {
		return "用户信息格式无效: " + err.Error()
	}
	if len([]rune(item.Group)) > 64 {
		return "分组长度不能超过 64 个字符"
	}
	if len([]rune(item.Tags)) > 255 {
		return "标签总长度不能超过 255 个字符"
	}
	if len([]rune(item.Remark)) > 255 {
		return "备注长度不能超过 255 个字符"
	}
	isStudent, err := parseUserImportBool(item.IsStudent)
	if err != nil {
		return "是否为学员仅支持填写 是/否"
	}
	if _, err := parseUserImportBool(item.IsAgent); err != nil {
		return "是否为代理仅支持填写 是/否"
	}
	rewardQuota, err := parseUserImportRewardQuota(item.Reward)
	if err != nil {
		return err.Error()
	}
	if rewardQuota > 0 && !isStudent {
		return "填写学员奖励金额时，是否为学员必须选择“是”"
	}

	nameTaken, err := model.IsUsernameTakenUnscoped(item.Username)
	if err != nil {
		return err.Error()
	}
	if nameTaken {
		return "用户名已存在"
	}
	email, err := model.NormalizeAndValidateAdminUserEmail(item.Email, 0)
	if err != nil {
		return err.Error()
	}
	phone, err := model.NormalizeAndValidateAdminUserPhone(item.Phone, 0)
	if err != nil {
		return err.Error()
	}
	item.Email = email
	item.Phone = phone

	seenUsernames[item.Username] = item.Row
	if item.Email != "" {
		seenEmails[strings.ToLower(item.Email)] = item.Row
	}
	if item.Phone != "" {
		seenPhones[item.Phone] = item.Row
	}
	return ""
}

func normalizeUserImportBoolText(value string) string {
	yes, err := parseUserImportBool(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	if yes {
		return "是"
	}
	return "否"
}

func parseUserImportBool(value string) (bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return false, nil
	}
	switch normalized {
	case "是", "yes", "y", "true", "1", "学员", "代理", "分销商", "oui", "да", "はい", "co", "có":
		return true, nil
	case "否", "no", "n", "false", "0", "non", "нет", "いいえ", "khong", "không":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool value")
	}
}

func parseUserImportRewardQuota(value string) (int, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return 0, nil
	}
	normalized = strings.ReplaceAll(normalized, ",", "")
	normalized = strings.ReplaceAll(normalized, "，", "")
	normalized = strings.ReplaceAll(normalized, "$", "")
	normalized = strings.ReplaceAll(normalized, "¥", "")
	normalized = strings.ReplaceAll(normalized, "￥", "")
	normalized = strings.TrimSpace(normalized)
	amount, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, fmt.Errorf("学员奖励金额格式不正确")
	}
	if amount < 0 {
		return 0, fmt.Errorf("学员奖励金额不能为负数")
	}
	if amount == 0 {
		return 0, nil
	}

	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	if rate <= 0 {
		rate = 1
	}
	return common.QuotaFromUSD(amount / rate), nil
}

func normalizeUserImportTags(tags string) string {
	parts := strings.FieldsFunc(tags, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '|' || r == '\n' || r == '\r'
	})
	return model.JoinUserTags(parts)
}

func buildUserImportWorkbook(failures []userImportFailure, locale userImportTemplateLocale) ([]byte, error) {
	f := excelize.NewFile()
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, locale.SheetName); err != nil {
		return nil, err
	}
	groupOptions := getUserImportGroupOptions()

	headers := buildUserImportWorkbookHeaders(locale)
	if len(failures) > 0 {
		headers = append([]string{}, headers...)
		headers = append(headers, locale.FailureReason)
	}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(locale.SheetName, cell, header); err != nil {
			return nil, err
		}
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	_ = f.SetCellStyle(locale.SheetName, "A1", cellName(len(headers), 1), headerStyle)
	_ = f.SetPanes(locale.SheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})
	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		width := 18.0
		if i == 6 || i == 8 || i == 10 || i == len(headers)-1 && len(failures) > 0 {
			width = 30
		}
		_ = f.SetColWidth(locale.SheetName, col, col, width)
	}

	for idx, failure := range failures {
		row := idx + 2
		values := []any{
			failure.Username,
			failure.Password,
			failure.DisplayName,
			failure.Email,
			failure.Phone,
			failure.Group,
			failure.Tags,
			localizeUserImportBoolText(failure.IsStudent, locale),
			failure.Reward,
			localizeUserImportBoolText(failure.IsAgent, locale),
			failure.Remark,
			failure.Reason,
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			if err := f.SetCellValue(locale.SheetName, cell, value); err != nil {
				return nil, err
			}
		}
	}

	if err := addUserImportDropdowns(f, locale.SheetName, groupOptions, locale); err != nil {
		return nil, err
	}

	if len(failures) == 0 {
		example := []any{
			"example_user",
			"password123",
			locale.ExampleUser,
			"example@example.com",
			"13800138000",
			"default",
			"VIP,测试",
			locale.Yes,
			"0",
			locale.No,
			locale.ExampleRemark,
		}
		for col, value := range example {
			cell, _ := excelize.CoordinatesToCellName(col+1, 2)
			if err := f.SetCellValue(locale.SheetName, cell, value); err != nil {
				return nil, err
			}
		}
		exampleStyle, _ := f.NewStyle(&excelize.Style{
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFF2CC"}, Pattern: 1},
			Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		})
		_ = f.SetCellStyle(locale.SheetName, "A2", cellName(len(headers), 2), exampleStyle)

	}

	if err := addUserImportInfoSheet(f, headerStyle, locale); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildUserImportWorkbookHeaders(locale userImportTemplateLocale) []string {
	headers := append([]string{}, locale.Headers[:8]...)
	headers = append(headers, userImportRewardHeader(locale))
	headers = append(headers, locale.Headers[8:]...)
	return headers
}

func userImportRewardHeader(locale userImportTemplateLocale) string {
	switch locale.Lang {
	case "zh-CN":
		return fmt.Sprintf("学员奖励金额(%s)", userImportRewardCurrencyCode())
	case "zh-TW":
		return fmt.Sprintf("學員獎勵金額(%s)", userImportRewardCurrencyCode())
	case "fr":
		return fmt.Sprintf("Montant recompense etudiant(%s)", userImportRewardCurrencyCode())
	case "ru":
		return fmt.Sprintf("Вознаграждение студента(%s)", userImportRewardCurrencyCode())
	case "ja":
		return fmt.Sprintf("学生報酬額(%s)", userImportRewardCurrencyCode())
	case "vi":
		return fmt.Sprintf("Thuong hoc vien(%s)", userImportRewardCurrencyCode())
	default:
		return fmt.Sprintf("Student Reward Amount(%s)", userImportRewardCurrencyCode())
	}
}

func userImportRewardCurrencyCode() string {
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		return "CNY"
	case operation_setting.QuotaDisplayTypeCustom:
		return "CUSTOM"
	default:
		return "USD"
	}
}

func userImportRewardNote(locale userImportTemplateLocale) []string {
	currency := userImportRewardCurrencyCode()
	switch locale.Lang {
	case "zh-CN":
		return []string{userImportRewardHeader(locale), fmt.Sprintf("选填，仅当“是否为学员”为“是”时生效；按当前站点展示币种 %s 填写，留空或 0 不赠送", currency)}
	case "zh-TW":
		return []string{userImportRewardHeader(locale), fmt.Sprintf("選填，僅當「是否為學員」為「是」時生效；按目前站點顯示幣種 %s 填寫，留空或 0 不贈送", currency)}
	case "fr":
		return []string{userImportRewardHeader(locale), fmt.Sprintf("Facultatif; actif uniquement si l'utilisateur est etudiant. Saisir en devise d'affichage du site %s; vide ou 0 = aucun cadeau", currency)}
	case "ru":
		return []string{userImportRewardHeader(locale), fmt.Sprintf("Необязательно; действует только если пользователь студент. Укажите в валюте отображения сайта %s; пусто или 0 = без подарка", currency)}
	case "ja":
		return []string{userImportRewardHeader(locale), fmt.Sprintf("任意。学生が「はい」の場合のみ有効。サイト表示通貨 %s で入力。空欄または 0 は付与なし", currency)}
	case "vi":
		return []string{userImportRewardHeader(locale), fmt.Sprintf("Tuy chon; chi co hieu luc khi La hoc vien la Co. Nhap theo tien te hien thi cua site %s; de trong hoac 0 thi khong tang", currency)}
	default:
		return []string{userImportRewardHeader(locale), fmt.Sprintf("Optional; only applies when Is Student is Yes. Enter in current site display currency %s; blank or 0 grants nothing", currency)}
	}
}

func buildUserImportNotes(locale userImportTemplateLocale) [][]string {
	notes := make([][]string, 0, len(locale.Notes)+1)
	for idx, note := range locale.Notes {
		if idx == len(locale.Notes)-1 {
			notes = append(notes, userImportRewardNote(locale))
		}
		notes = append(notes, note)
	}
	return notes
}

func addUserImportInfoSheet(f *excelize.File, headerStyle int, locale userImportTemplateLocale) error {
	infoSheet := locale.InfoSheetName
	if _, err := f.NewSheet(infoSheet); err != nil {
		return err
	}
	for r, row := range buildUserImportNotes(locale) {
		for cidx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(cidx+1, r+1)
			if err := f.SetCellValue(infoSheet, cell, value); err != nil {
				return err
			}
		}
	}
	_ = f.SetCellStyle(infoSheet, "A1", "B1", headerStyle)
	_ = f.SetColWidth(infoSheet, "A", "A", 18)
	_ = f.SetColWidth(infoSheet, "B", "B", 70)
	return nil
}

func getUserImportGroupOptions() []string {
	groupSet := map[string]struct{}{"default": {}}
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupName = strings.TrimSpace(groupName)
		if groupName != "" {
			groupSet[groupName] = struct{}{}
		}
	}

	groups := make([]string, 0, len(groupSet))
	for groupName := range groupSet {
		if groupName != "default" {
			groups = append(groups, groupName)
		}
	}
	sort.Strings(groups)
	return append([]string{"default"}, groups...)
}

func localizeUserImportBoolText(value string, locale userImportTemplateLocale) string {
	yes, err := parseUserImportBool(value)
	if err != nil {
		return strings.TrimSpace(value)
	}
	if yes {
		return locale.Yes
	}
	return locale.No
}

func addUserImportDropdowns(f *excelize.File, sheet string, groupOptions []string, locale userImportTemplateLocale) error {
	passwordDV := excelize.NewDataValidation(true)
	passwordDV.SetSqref("B2:B1000")
	if err := passwordDV.SetRange(8, 20, excelize.DataValidationTypeTextLength, excelize.DataValidationOperatorBetween); err != nil {
		return err
	}
	passwordDV.SetInput(locale.PasswordInputTitle, locale.PasswordInputMessage)
	passwordDV.SetError(excelize.DataValidationErrorStyleWarning, locale.PasswordWarningTitle, locale.PasswordWarningMsg)
	if err := f.AddDataValidation(sheet, passwordDV); err != nil {
		return err
	}

	rewardDV := excelize.NewDataValidation(true)
	rewardDV.SetSqref("I2:I1000")
	if err := rewardDV.SetRange(0, 1000000000, excelize.DataValidationTypeDecimal, excelize.DataValidationOperatorBetween); err != nil {
		return err
	}
	rewardDV.SetInput(userImportRewardHeader(locale), userImportRewardNote(locale)[1])
	rewardDV.SetError(excelize.DataValidationErrorStyleWarning, userImportRewardHeader(locale), "请输入大于等于 0 的金额")
	if err := f.AddDataValidation(sheet, rewardDV); err != nil {
		return err
	}

	if len(groupOptions) > 0 {
		optionSheet := locale.OptionsSheetName
		if _, err := f.NewSheet(optionSheet); err != nil {
			return err
		}
		for idx, groupName := range groupOptions {
			cell, _ := excelize.CoordinatesToCellName(1, idx+1)
			if err := f.SetCellValue(optionSheet, cell, groupName); err != nil {
				return err
			}
		}
		_ = f.SetColWidth(optionSheet, "A", "A", 24)
		if err := f.SetSheetVisible(optionSheet, false, true); err != nil {
			return err
		}

		dv := excelize.NewDataValidation(true)
		dv.SetSqref("F2:F1000")
		dv.SetSqrefDropList(fmt.Sprintf("'%s'!$A$1:$A$%d", optionSheet, len(groupOptions)))
		dv.SetInput(locale.GroupInputTitle, locale.GroupInputMessage)
		if err := f.AddDataValidation(sheet, dv); err != nil {
			return err
		}
	}

	for _, cellRange := range []string{"H2:H1000", "J2:J1000"} {
		dv := excelize.NewDataValidation(true)
		dv.SetSqref(cellRange)
		if err := dv.SetDropList([]string{locale.No, locale.Yes}); err != nil {
			return err
		}
		dv.SetInput(locale.BoolInputTitle, locale.BoolInputMessage)
		if err := f.AddDataValidation(sheet, dv); err != nil {
			return err
		}
	}
	return nil
}

func cellName(col int, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func writeUserImportWorkbook(c *gin.Context, data []byte, filename string) {
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func storeUserImportFailureFile(id string, data []byte) {
	userImportFailureFilesMu.Lock()
	defer userImportFailureFilesMu.Unlock()
	now := time.Now()
	for key, value := range userImportFailureFiles {
		if now.Sub(value.createdAt) > time.Hour {
			delete(userImportFailureFiles, key)
		}
	}
	userImportFailureFiles[id] = userImportFailureFile{bytes: data, createdAt: now}
}

func getUserImportFailureFile(id string) ([]byte, bool) {
	userImportFailureFilesMu.Lock()
	defer userImportFailureFilesMu.Unlock()
	value, ok := userImportFailureFiles[id]
	if !ok || time.Since(value.createdAt) > time.Hour {
		delete(userImportFailureFiles, id)
		return nil, false
	}
	return value.bytes, true
}
