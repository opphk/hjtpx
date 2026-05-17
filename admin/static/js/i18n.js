// 国际化支持 - 管理后台增强版
const I18n = {
    currentLang: 'zh-CN',
    currentTimezone: 'Asia/Shanghai',
    translations: {},
    localeData: {},
    
    supportedLangs: [
        { code: 'zh-CN', name: '简体中文', nativeName: '简体中文', flag: '🇨🇳', region: 'Asia' },
        { code: 'en-US', name: 'English (US)', nativeName: 'English', flag: '🇺🇸', region: 'Americas' },
        { code: 'ja-JP', name: 'Japanese', nativeName: '日本語', flag: '🇯🇵', region: 'Asia' },
        { code: 'ko-KR', name: 'Korean', nativeName: '한국어', flag: '🇰🇷', region: 'Asia' },
        { code: 'fr-FR', name: 'French', nativeName: 'Français', flag: '🇫🇷', region: 'Europe' },
        { code: 'de-DE', name: 'German', nativeName: 'Deutsch', flag: '🇩🇪', region: 'Europe' },
        { code: 'es-ES', name: 'Spanish', nativeName: 'Español', flag: '🇪🇸', region: 'Europe' },
        { code: 'pt-BR', name: 'Portuguese (Brazil)', nativeName: 'Português (Brasil)', flag: '🇧🇷', region: 'Americas' },
        { code: 'it-IT', name: 'Italian', nativeName: 'Italiano', flag: '🇮🇹', region: 'Europe' },
        { code: 'ru-RU', name: 'Russian', nativeName: 'Русский', flag: '🇷🇺', region: 'Europe' },
        { code: 'ar-SA', name: 'Arabic', nativeName: 'العربية', flag: '🇸🇦', region: 'Middle East', rtl: true },
        { code: 'th-TH', name: 'Thai', nativeName: 'ไทย', flag: '🇹🇭', region: 'Asia' },
        { code: 'vi-VN', name: 'Vietnamese', nativeName: 'Tiếng Việt', flag: '🇻🇳', region: 'Asia' },
        { code: 'id-ID', name: 'Indonesian', nativeName: 'Bahasa Indonesia', flag: '🇮🇩', region: 'Asia' },
        { code: 'ms-MY', name: 'Malay', nativeName: 'Bahasa Melayu', flag: '🇲🇾', region: 'Asia' },
        { code: 'tl-PH', name: 'Filipino', nativeName: 'Filipino', flag: '🇵🇭', region: 'Asia' },
        { code: 'fa-IR', name: 'Persian (Iran)', nativeName: 'فارسی', flag: '🇮🇷', region: 'Middle East', rtl: true },
        { code: 'he-IL', name: 'Hebrew', nativeName: 'עברית', flag: '🇮🇱', region: 'Middle East', rtl: true },
        { code: 'tr-TR', name: 'Turkish', nativeName: 'Türkçe', flag: '🇹🇷', region: 'Europe' },
        { code: 'pl-PL', name: 'Polish', nativeName: 'Polski', flag: '🇵🇱', region: 'Europe' },
        { code: 'nl-NL', name: 'Dutch', nativeName: 'Nederlands', flag: '🇳🇱', region: 'Europe' },
        { code: 'el-GR', name: 'Greek', nativeName: 'Ελληνικά', flag: '🇬🇷', region: 'Europe' },
        { code: 'cs-CZ', name: 'Czech', nativeName: 'Čeština', flag: '🇨🇿', region: 'Europe' },
        { code: 'sv-SE', name: 'Swedish', nativeName: 'Svenska', flag: '🇸🇪', region: 'Europe' },
        { code: 'da-DK', name: 'Danish', nativeName: 'Dansk', flag: '🇩🇰', region: 'Europe' },
        { code: 'fi-FI', name: 'Finnish', nativeName: 'Suomi', flag: '🇫🇮', region: 'Europe' },
        { code: 'no-NO', name: 'Norwegian', nativeName: 'Norsk', flag: '🇳🇴', region: 'Europe' },
        { code: 'hu-HU', name: 'Hungarian', nativeName: 'Magyar', flag: '🇭🇺', region: 'Europe' },
        { code: 'ro-RO', name: 'Romanian', nativeName: 'Română', flag: '🇷🇴', region: 'Europe' },
        { code: 'uk-UA', name: 'Ukrainian', nativeName: 'Українська', flag: '🇺🇦', region: 'Europe' },
        { code: 'bg-BG', name: 'Bulgarian', nativeName: 'Български', flag: '🇧🇬', region: 'Europe' },
        { code: 'hr-HR', name: 'Croatian', nativeName: 'Hrvatski', flag: '🇭🇷', region: 'Europe' },
        { code: 'sk-SK', name: 'Slovak', nativeName: 'Slovenčina', flag: '🇸🇰', region: 'Europe' },
        { code: 'sl-SI', name: 'Slovenian', nativeName: 'Slovenščina', flag: '🇸🇮', region: 'Europe' }
    ],

    supportedTimezones: [
        { id: 'Asia/Shanghai', name: 'China Standard Time', offset: '+08:00' },
        { id: 'Asia/Tokyo', name: 'Japan Standard Time', offset: '+09:00' },
        { id: 'Asia/Seoul', name: 'Korea Standard Time', offset: '+09:00' },
        { id: 'Asia/Singapore', name: 'Singapore Time', offset: '+08:00' },
        { id: 'Asia/Hong_Kong', name: 'Hong Kong Time', offset: '+08:00' },
        { id: 'Asia/Dubai', name: 'Gulf Standard Time', offset: '+04:00' },
        { id: 'Asia/Jerusalem', name: 'Israel Standard Time', offset: '+02:00' },
        { id: 'Asia/Kolkata', name: 'India Standard Time', offset: '+05:30' },
        { id: 'Asia/Bangkok', name: 'Indochina Time', offset: '+07:00' },
        { id: 'Asia/Jakarta', name: 'Western Indonesia Time', offset: '+07:00' },
        { id: 'Asia/Manila', name: 'Philippine Time', offset: '+08:00' },
        { id: 'Asia/Tehran', name: 'Iran Standard Time', offset: '+03:30' },
        { id: 'Europe/London', name: 'Greenwich Mean Time', offset: '+00:00' },
        { id: 'Europe/Paris', name: 'Central European Time', offset: '+01:00' },
        { id: 'Europe/Berlin', name: 'Central European Time', offset: '+01:00' },
        { id: 'Europe/Moscow', name: 'Moscow Standard Time', offset: '+03:00' },
        { id: 'Europe/Istanbul', name: 'Turkey Time', offset: '+03:00' },
        { id: 'Europe/Warsaw', name: 'Central European Time', offset: '+01:00' },
        { id: 'Europe/Athens', name: 'Eastern European Time', offset: '+02:00' },
        { id: 'America/New_York', name: 'Eastern Standard Time', offset: '-05:00' },
        { id: 'America/Los_Angeles', name: 'Pacific Standard Time', offset: '-08:00' },
        { id: 'America/Chicago', name: 'Central Standard Time', offset: '-06:00' },
        { id: 'America/Denver', name: 'Mountain Standard Time', offset: '-07:00' },
        { id: 'America/Sao_Paulo', name: 'Brasília Time', offset: '-03:00' },
        { id: 'America/Mexico_City', name: 'Central Standard Time (Mexico)', offset: '-06:00' },
        { id: 'America/Toronto', name: 'Eastern Standard Time (Canada)', offset: '-05:00' },
        { id: 'America/Vancouver', name: 'Pacific Standard Time (Canada)', offset: '-08:00' },
        { id: 'Australia/Sydney', name: 'Australian Eastern Standard Time', offset: '+10:00' },
        { id: 'Australia/Melbourne', name: 'Australian Eastern Standard Time', offset: '+10:00' },
        { id: 'Australia/Perth', name: 'Australian Western Standard Time', offset: '+08:00' },
        { id: 'Pacific/Auckland', name: 'New Zealand Standard Time', offset: '+12:00' },
        { id: 'Pacific/Honolulu', name: 'Hawaii Standard Time', offset: '-10:00' },
        { id: 'UTC', name: 'Coordinated Universal Time', offset: '+00:00' }
    ],

    localeConfigs: {
        'zh-CN': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: '¥', firstWeekday: 1 },
        'en-US': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: '$', firstWeekday: 0 },
        'ja-JP': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: '¥', firstWeekday: 0 },
        'ko-KR': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: '₩', firstWeekday: 0 },
        'fr-FR': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: '€', firstWeekday: 1 },
        'de-DE': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '€', firstWeekday: 1 },
        'es-ES': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '€', firstWeekday: 1 },
        'pt-BR': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: 'R$', firstWeekday: 0 },
        'it-IT': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '€', firstWeekday: 1 },
        'ru-RU': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: '₽', firstWeekday: 1 },
        'ar-SA': { decimalSeparator: '٫', thousandSeparator: '٬', currencySymbol: 'ر.س', firstWeekday: 6 },
        'th-TH': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: '฿', firstWeekday: 0 },
        'vi-VN': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '₫', firstWeekday: 0 },
        'id-ID': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: 'Rp', firstWeekday: 0 },
        'ms-MY': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: 'RM', firstWeekday: 0 },
        'tl-PH': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: '₱', firstWeekday: 0 },
        'fa-IR': { decimalSeparator: '٫', thousandSeparator: '٬', currencySymbol: 'ریال', firstWeekday: 5 },
        'he-IL': { decimalSeparator: '.', thousandSeparator: ',', currencySymbol: '₪', firstWeekday: 0 },
        'tr-TR': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '₺', firstWeekday: 1 },
        'pl-PL': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: 'zł', firstWeekday: 1 },
        'nl-NL': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '€', firstWeekday: 1 },
        'el-GR': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '€', firstWeekday: 1 },
        'cs-CZ': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: 'Kč', firstWeekday: 1 },
        'sv-SE': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: 'kr', firstWeekday: 1 },
        'da-DK': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: 'kr', firstWeekday: 1 },
        'fi-FI': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: '€', firstWeekday: 1 },
        'no-NO': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: 'kr', firstWeekday: 1 },
        'hu-HU': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: 'Ft', firstWeekday: 1 },
        'ro-RO': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: 'lei', firstWeekday: 1 },
        'uk-UA': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: '₴', firstWeekday: 1 },
        'bg-BG': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: 'лв', firstWeekday: 1 },
        'hr-HR': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '€', firstWeekday: 1 },
        'sk-SK': { decimalSeparator: ',', thousandSeparator: ' ', currencySymbol: '€', firstWeekday: 1 },
        'sl-SI': { decimalSeparator: ',', thousandSeparator: '.', currencySymbol: '€', firstWeekday: 1 }
    },

    monthNames: {
        'zh-CN': ['一月', '二月', '三月', '四月', '五月', '六月', '七月', '八月', '九月', '十月', '十一月', '十二月'],
        'en-US': ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'],
        'ja-JP': ['1月', '2月', '3月', '4月', '5月', '6月', '7月', '8月', '9月', '10月', '11月', '12月'],
        'ko-KR': ['1월', '2월', '3월', '4월', '5월', '6월', '7월', '8월', '9월', '10월', '11월', '12월'],
        'fr-FR': ['janvier', 'février', 'mars', 'avril', 'mai', 'juin', 'juillet', 'août', 'septembre', 'octobre', 'novembre', 'décembre'],
        'de-DE': ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'],
        'es-ES': ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'],
        'pt-BR': ['janeiro', 'fevereiro', 'março', 'abril', 'maio', 'junho', 'julho', 'agosto', 'setembro', 'outubro', 'novembro', 'dezembro'],
        'it-IT': ['gennaio', 'febbraio', 'marzo', 'aprile', 'maggio', 'giugno', 'luglio', 'agosto', 'settembre', 'ottobre', 'novembre', 'dicembre'],
        'ru-RU': ['января', 'февраля', 'марта', 'апреля', 'мая', 'июня', 'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря'],
        'ar-SA': ['يناير', 'فبراير', 'مارس', 'أبريل', 'مايو', 'يونيو', 'يوليو', 'أغسطس', 'سبتمبر', 'أكتوبر', 'نوفمبر', 'ديسمبر'],
        'th-TH': ['มกราคม', 'กุมภาพันธ์', 'มีนาคม', 'เมษายน', 'พฤษภาคม', 'มิถุนายน', 'กรกฎาคม', 'สิงหาคม', 'กันยายน', 'ตุลาคม', 'พฤศจิกายน', 'ธันวาคม'],
        'vi-VN': ['Tháng 1', 'Tháng 2', 'Tháng 3', 'Tháng 4', 'Tháng 5', 'Tháng 6', 'Tháng 7', 'Tháng 8', 'Tháng 9', 'Tháng 10', 'Tháng 11', 'Tháng 12'],
        'id-ID': ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember'],
        'ms-MY': ['Januari', 'Februari', 'Mac', 'April', 'Mei', 'Jun', 'Julai', 'Ogos', 'September', 'Oktober', 'November', 'Disember'],
        'tl-PH': ['Enero', 'Pebrero', 'Marso', 'Abril', 'Mayo', 'Hunyo', 'Hulyo', 'Agosto', 'Setyembre', 'Oktubre', 'Nobyembre', 'Disyembre'],
        'fa-IR': ['ژانویه', 'فوریه', 'مارس', 'آوریل', 'مه', 'ژوئن', 'ژوئیه', 'اوت', 'سپتامبر', 'اکتبر', 'نوامبر', 'دسامبر'],
        'he-IL': ['ינואר', 'פברואר', 'מרץ', 'אפריל', 'מאי', 'יוני', 'יולי', 'אוגוסט', 'ספטמבר', 'אוקטובר', 'נובמבר', 'דצמבר'],
        'tr-TR': ['Ocak', 'Şubat', 'Mart', 'Nisan', 'Mayıs', 'Haziran', 'Temmuz', 'Ağustos', 'Eylül', 'Ekim', 'Kasım', 'Aralık'],
        'pl-PL': ['stycznia', 'lutego', 'marca', 'kwietnia', 'maja', 'czerwca', 'lipca', 'sierpnia', 'września', 'października', 'listopada', 'grudnia'],
        'nl-NL': ['januari', 'februari', 'maart', 'april', 'mei', 'juni', 'juli', 'augustus', 'september', 'oktober', 'november', 'december'],
        'el-GR': ['Ιανουαρίου', 'Φεβρουαρίου', 'Μαρτίου', 'Απριλίου', 'Μαΐου', 'Ιουνίου', 'Ιουλίου', 'Αυγούστου', 'Σεπτεμβρίου', 'Οκτωβρίου', 'Νοεμβρίου', 'Δεκεμβρίου'],
        'cs-CZ': ['ledna', 'února', 'března', 'dubna', 'května', 'června', 'července', 'srpna', 'září', 'října', 'listopadu', 'prosince'],
        'sv-SE': ['januari', 'februari', 'mars', 'april', 'maj', 'juni', 'juli', 'augusti', 'september', 'oktober', 'november', 'december'],
        'da-DK': ['januar', 'februar', 'marts', 'april', 'maj', 'juni', 'juli', 'august', 'september', 'oktober', 'november', 'december'],
        'fi-FI': ['tammikuuta', 'helmikuuta', 'maaliskuuta', 'huhtikuuta', 'toukokuuta', 'kesäkuuta', 'heinäkuuta', 'elokuuta', 'syyskuuta', 'lokakuuta', 'marraskuuta', 'joulukuuta'],
        'no-NO': ['januar', 'februar', 'mars', 'april', 'mai', 'juni', 'juli', 'august', 'september', 'oktober', 'november', 'desember'],
        'hu-HU': ['január', 'február', 'március', 'április', 'május', 'június', 'július', 'augusztus', 'szeptember', 'október', 'november', 'december'],
        'ro-RO': ['ianuarie', 'februarie', 'martie', 'aprilie', 'mai', 'iunie', 'iulie', 'august', 'septembrie', 'octombrie', 'noiembrie', 'decembrie'],
        'uk-UA': ['січня', 'лютого', 'березня', 'квітня', 'травня', 'червня', 'липня', 'серпня', 'вересня', 'жовтня', 'листопада', 'грудня'],
        'bg-BG': ['януари', 'февруари', 'март', 'април', 'май', 'юни', 'юли', 'август', 'септември', 'октомври', 'ноември', 'декември'],
        'hr-HR': ['siječnja', 'veljače', 'ožujka', 'travnja', 'svibnja', 'lipnja', 'srpnja', 'kolovoza', 'rujna', 'listopada', 'studenog', 'prosinca'],
        'sk-SK': ['januára', 'februára', 'marca', 'apríla', 'mája', 'júna', 'júla', 'augusta', 'septembra', 'októbra', 'novembra', 'decembra'],
        'sl-SI': ['januarja', 'februarja', 'marca', 'aprila', 'maja', 'junija', 'julija', 'avgusta', 'septembra', 'oktobra', 'novembra', 'decembra']
    },

    weekdayNames: {
        'zh-CN': ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'],
        'en-US': ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'],
        'ja-JP': ['日', '月', '火', '水', '木', '金', '土'],
        'ko-KR': ['일', '월', '화', '수', '목', '금', '토'],
        'fr-FR': ['dimanche', 'lundi', 'mardi', 'mercredi', 'jeudi', 'vendredi', 'samedi'],
        'de-DE': ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag'],
        'es-ES': ['domingo', 'lunes', 'martes', 'miércoles', 'jueves', 'viernes', 'sábado'],
        'pt-BR': ['domingo', 'segunda-feira', 'terça-feira', 'quarta-feira', 'quinta-feira', 'sexta-feira', 'sábado'],
        'it-IT': ['domenica', 'lunedì', 'martedì', 'mercoledì', 'giovedì', 'venerdì', 'sabato'],
        'ru-RU': ['воскресенье', 'понедельник', 'вторник', 'среда', 'четверг', 'пятница', 'суббота'],
        'ar-SA': ['الأحد', 'الاثنين', 'الثلاثاء', 'الأربعاء', 'الخميس', 'الجمعة', 'السبت'],
        'th-TH': ['อาทิตย์', 'จันทร์', 'อังคาร', 'พุธ', 'พฤหัสบดี', 'ศุกร์', 'เสาร์'],
        'vi-VN': ['Chủ Nhật', 'Thứ Hai', 'Thứ Ba', 'Thứ Tư', 'Thứ Năm', 'Thứ Sáu', 'Thứ Bảy'],
        'id-ID': ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'],
        'ms-MY': ['Ahad', 'Isnin', 'Selasa', 'Rabu', 'Kamis', 'Jumaat', 'Sabtu'],
        'tl-PH': ['Linggo', 'Lunes', 'Martes', 'Miyerkules', 'Huwebes', 'Biyernes', 'Sabado'],
        'fa-IR': ['یکشنبه', 'دوشنبه', 'سه‌شنبه', 'چهارشنبه', 'پنجشنبه', 'جمعه', 'شنبه'],
        'he-IL': ['יום ראשון', 'יום שני', 'יום שלישי', 'יום רביעי', 'יום חמישי', 'יום שישי', 'שבת'],
        'tr-TR': ['Pazar', 'Pazartesi', 'Salı', 'Çarşamba', 'Perşembe', 'Cuma', 'Cumartesi'],
        'pl-PL': ['niedziela', 'poniedziałek', 'wtorek', 'środa', 'czwartek', 'piątek', 'sobota'],
        'nl-NL': ['zondag', 'maandag', 'dinsdag', 'woensdag', 'donderdag', 'vrijdag', 'zaterdag'],
        'el-GR': ['Κυριακή', 'Δευτέρα', 'Τρίτη', 'Τετάρτη', 'Πέμπτη', 'Παρασκευή', 'Σάββατο'],
        'cs-CZ': ['neděle', 'pondělí', 'úterý', 'středa', 'čtvrtek', 'pátek', 'sobota'],
        'sv-SE': ['söndag', 'måndag', 'tisdag', 'onsdag', 'torsdag', 'fredag', 'lördag'],
        'da-DK': ['søndag', 'mandag', 'tirsdag', 'onsdag', 'torsdag', 'fredag', 'lørdag'],
        'fi-FI': ['sunnuntai', 'maanantai', 'tiistai', 'keskiviikko', 'torstai', 'perjantai', 'lauantai'],
        'no-NO': ['søndag', 'mandag', 'tirsdag', 'onsdag', 'torsdag', 'fredag', 'lørdag'],
        'hu-HU': ['vasárnap', 'hétfő', 'kedd', 'szerda', 'csütörtök', 'péntek', 'szombat'],
        'ro-RO': ['duminică', 'luni', 'marți', 'miercuri', 'joi', 'vineri', 'sâmbătă'],
        'uk-UA': ['неділя', 'понеділок', 'вівторок', 'середа', 'четвер', 'п\'ятниця', 'субота'],
        'bg-BG': ['неделя', 'понеделник', 'вторник', 'сряда', 'четвъртък', 'петък', 'събота'],
        'hr-HR': ['nedjelja', 'ponedjeljak', 'utorak', 'srijeda', 'četvrtak', 'petak', 'subota'],
        'sk-SK': ['nedeľa', 'pondelok', 'utorok', 'streda', 'štvrtok', 'piatok', 'sobota'],
        'sl-SI': ['nedelja', 'ponedeljek', 'torek', 'sreda', 'četrtek', 'petek', 'sobota']
    },

    getBrowserLang: function() {
        const navLang = navigator.language || navigator.userLanguage;
        for (let lang of this.supportedLangs) {
            if (navLang.startsWith(lang.code.split('-')[0])) {
                return lang.code;
            }
        }
        return 'zh-CN';
    },

    getStoredLang: function() {
        return localStorage.getItem('adminPreferredLang');
    },

    setStoredLang: function(lang) {
        localStorage.setItem('adminPreferredLang', lang);
    },

    getStoredTimezone: function() {
        return localStorage.getItem('adminPreferredTimezone') || Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai';
    },

    setStoredTimezone: function(tz) {
        localStorage.setItem('adminPreferredTimezone', tz);
    },

    init: async function() {
        this.currentLang = this.getStoredLang() || this.getBrowserLang();
        this.currentTimezone = this.getStoredTimezone();
        
        await this.loadTranslations();
        
        this.renderLangSelector();
        this.renderTimezoneSelector();
        
        this.applyTranslations();
        
        document.documentElement.lang = this.currentLang;
        document.documentElement.dir = this.getLangInfo().rtl ? 'rtl' : 'ltr';
    },

    loadTranslations: async function() {
        try {
            const zhCN = await fetch(`/admin/translations/zh-CN.json`);
            this.translations['zh-CN'] = await zhCN.json();
            
            if (this.currentLang !== 'zh-CN') {
                try {
                    const target = await fetch(`/admin/translations/${this.currentLang}.json`);
                    this.translations[this.currentLang] = await target.json();
                } catch (e) {
                    console.warn('Failed to load target language, using default');
                }
            }
        } catch (e) {
            console.error('Failed to load translations:', e);
        }
    },

    t: function(key, params = {}) {
        let text = this.translations[this.currentLang]?.[key] || 
                   this.translations['zh-CN']?.[key] || key;
        
        Object.keys(params).forEach(paramKey => {
            text = text.replace(`{${paramKey}}`, params[paramKey]);
        });
        
        return text;
    },

    getLangInfo: function() {
        return this.supportedLangs.find(l => l.code === this.currentLang) || this.supportedLangs[0];
    },

    applyTranslations: function() {
        document.querySelectorAll('[data-i18n]').forEach(el => {
            const key = el.getAttribute('data-i18n');
            el.textContent = this.t(key);
        });
        
        document.querySelectorAll('[data-i18n-placeholder]').forEach(el => {
            const key = el.getAttribute('data-i18n-placeholder');
            el.placeholder = this.t(key);
        });
        
        document.querySelectorAll('[data-i18n-title]').forEach(el => {
            const key = el.getAttribute('data-i18n-title');
            el.title = this.t(key);
        });
        
        const titleKey = document.documentElement.getAttribute('data-i18n-title');
        if (titleKey) {
            document.title = this.t(titleKey);
        }
    },

    renderLangSelector: function() {
        let container = document.getElementById('admin-lang-selector-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'admin-lang-selector-container';
            
            const headerRight = document.querySelector('.top-header-right');
            if (headerRight) {
                container.style.cssText = 'display: inline-block; margin-right: 10px;';
                headerRight.insertBefore(container, headerRight.firstChild);
            } else {
                container.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999;';
                document.body.appendChild(container);
            }
        }
        
        container.innerHTML = '';
        
        const select = document.createElement('select');
        select.id = 'admin-lang-selector';
        select.style.cssText = 'padding: 4px 8px; border: 1px solid #ddd; border-radius: 4px; background: white; font-size: 13px; cursor: pointer;';
        
        this.supportedLangs.forEach(lang => {
            const option = document.createElement('option');
            option.value = lang.code;
            option.textContent = `${lang.flag} ${lang.nativeName}`;
            if (lang.code === this.currentLang) {
                option.selected = true;
            }
            select.appendChild(option);
        });
        
        select.addEventListener('change', (e) => {
            this.setLang(e.target.value);
        });
        
        container.appendChild(select);
    },

    renderTimezoneSelector: function() {
        let container = document.getElementById('admin-timezone-selector-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'admin-timezone-selector-container';
            
            const headerRight = document.querySelector('.top-header-right');
            if (headerRight) {
                container.style.cssText = 'display: inline-block; margin-right: 10px;';
                headerRight.insertBefore(container, headerRight.firstChild);
            } else {
                container.style.cssText = 'position: fixed; top: 60px; right: 20px; z-index: 9999;';
                document.body.appendChild(container);
            }
        }
        
        container.innerHTML = '';
        
        const select = document.createElement('select');
        select.id = 'admin-timezone-selector';
        select.style.cssText = 'padding: 4px 8px; border: 1px solid #ddd; border-radius: 4px; background: white; font-size: 13px; cursor: pointer;';
        
        this.supportedTimezones.forEach(tz => {
            const option = document.createElement('option');
            option.value = tz.id;
            option.textContent = `${tz.name} (${tz.offset})`;
            if (tz.id === this.currentTimezone) {
                option.selected = true;
            }
            select.appendChild(option);
        });
        
        select.addEventListener('change', (e) => {
            this.setTimezone(e.target.value);
        });
        
        container.appendChild(select);
    },

    setLang: async function(lang) {
        if (lang === this.currentLang) return;
        
        this.currentLang = lang;
        this.setStoredLang(lang);
        
        if (!this.translations[lang]) {
            try {
                const response = await fetch(`/admin/translations/${lang}.json`);
                this.translations[lang] = await response.json();
            } catch (e) {
                console.error('Failed to load new language:', e);
            }
        }
        
        this.applyTranslations();
        document.documentElement.lang = lang;
        document.documentElement.dir = this.getLangInfo().rtl ? 'rtl' : 'ltr';
        
        document.dispatchEvent(new CustomEvent('adminLanguageChange', { detail: { lang } }));
    },

    setTimezone: function(tz) {
        this.currentTimezone = tz;
        this.setStoredTimezone(tz);
        document.dispatchEvent(new CustomEvent('adminTimezoneChange', { detail: { timezone: tz } }));
    },

    formatNumber: function(num, options = {}) {
        const config = this.localeConfigs[this.currentLang] || this.localeConfigs['en-US'];
        const { decimalPlaces = 2, useThousandSeparator = true } = options;
        
        const absNum = Math.abs(num);
        const intPart = Math.floor(absNum);
        const decPart = absNum - intPart;
        
        let intStr = intPart.toString();
        if (useThousandSeparator) {
            intStr = intStr.replace(/\B(?=(\d{3})+(?!\d))/g, config.thousandSeparator);
        }
        
        let result = intStr;
        if (decimalPlaces > 0) {
            result += config.decimalSeparator + decPart.toFixed(decimalPlaces).slice(2);
        }
        
        return num < 0 ? '-' + result : result;
    },

    formatCurrency: function(amount, options = {}) {
        const config = this.localeConfigs[this.currentLang] || this.localeConfigs['en-US'];
        const { currencyCode = 'USD', showCode = false } = options;
        
        const formatted = this.formatNumber(amount);
        const symbol = config.currencySymbol;
        
        if (showCode) {
            return `${symbol}${formatted} ${currencyCode}`;
        }
        return `${symbol}${formatted}`;
    },

    formatDate: function(date, options = {}) {
        const { style = 'medium' } = options;
        const d = new Date(date);
        const months = this.monthNames[this.currentLang] || this.monthNames['en-US'];
        const weekdays = this.weekdayNames[this.currentLang] || this.weekdayNames['en-US'];
        
        const year = d.getFullYear();
        const month = d.getMonth();
        const day = d.getDate();
        const weekday = d.getDay();
        
        const monthName = months[month];
        const weekdayName = weekdays[weekday];
        
        switch (style) {
            case 'short':
                return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
            case 'long':
                return `${weekdayName}, ${day} ${monthName} ${year}`;
            case 'medium':
            default:
                if (this.currentLang === 'en-US') {
                    return `${monthName} ${day}, ${year}`;
                } else if (this.currentLang === 'zh-CN' || this.currentLang === 'ja-JP' || this.currentLang === 'ko-KR') {
                    return `${year}年${month + 1}月${day}日`;
                } else if (this.currentLang === 'de-DE' || this.currentLang === 'nl-NL') {
                    return `${day}. ${monthName} ${year}`;
                } else if (this.currentLang === 'fr-FR' || this.currentLang === 'es-ES' || this.currentLang === 'it-IT' || this.currentLang === 'pt-BR') {
                    return `${day} ${monthName} ${year}`;
                } else {
                    return `${day} ${monthName} ${year}`;
                }
        }
    },

    formatTime: function(date, options = {}) {
        const { use24Hour = true } = options;
        const d = new Date(date);
        
        let hours = d.getHours();
        const minutes = String(d.getMinutes()).padStart(2, '0');
        const seconds = String(d.getSeconds()).padStart(2, '0');
        
        if (use24Hour) {
            return `${String(hours).padStart(2, '0')}:${minutes}:${seconds}`;
        } else {
            const ampm = hours >= 12 ? 'PM' : 'AM';
            hours = hours % 12 || 12;
            return `${hours}:${minutes}:${seconds} ${ampm}`;
        }
    },

    formatDateTime: function(date, options = {}) {
        const { dateStyle = 'medium', timeStyle = 'medium' } = options;
        return `${this.formatDate(date, { style: dateStyle })} ${this.formatTime(date, { use24Hour: this.currentLang !== 'en-US' })}`;
    },

    formatRelativeTime: function(date) {
        const d = new Date(date);
        const now = new Date();
        const diff = now - d;
        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        
        const langInfo = this.getLangInfo();
        
        if (Math.abs(diff) < 60000) {
            return this.t('just_now') || (langInfo.code === 'en-US' ? 'just now' : '刚刚');
        } else if (minutes < 60) {
            return this.t('minutes_ago', { n: minutes }) || `${minutes} ${langInfo.code === 'en-US' ? 'minutes ago' : '分钟前'}`;
        } else if (hours < 24) {
            return this.t('hours_ago', { n: hours }) || `${hours} ${langInfo.code === 'en-US' ? 'hours ago' : '小时前'}`;
        } else if (days < 7) {
            return this.t('days_ago', { n: days }) || `${days} ${langInfo.code === 'en-US' ? 'days ago' : '天前'}`;
        } else {
            return this.formatDate(d);
        }
    },

    formatDuration: function(milliseconds) {
        const seconds = Math.floor(milliseconds / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        
        if (days > 0) {
            return `${days}d ${hours % 24}h ${minutes % 60}m`;
        } else if (hours > 0) {
            return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        } else {
            return `${seconds}s`;
        }
    },

    getTimezoneOffset: function() {
        const tz = this.supportedTimezones.find(t => t.id === this.currentTimezone);
        return tz ? tz.offset : '+08:00';
    },

    convertTimezone: function(date, fromTz, toTz) {
        const d = new Date(date);
        return new Date(d.toLocaleString('en-US', { timeZone: toTz }));
    }
};

document.addEventListener('DOMContentLoaded', function() {
    I18n.init();
});
