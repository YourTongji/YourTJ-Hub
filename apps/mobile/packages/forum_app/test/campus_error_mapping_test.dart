import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/l10n/app_localizations_de.dart';
import 'package:forum_app/l10n/app_localizations_en.dart';
import 'package:forum_app/l10n/app_localizations_ja.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/pages/campus/campus_helpers.dart';

ApiException _api(String messageCode) =>
    ApiException(fallbackMessage: messageCode, messageCode: messageCode);

void main() {
  final locales = <AppLocalizations>[
    AppLocalizationsZh(),
    AppLocalizationsEn(),
    AppLocalizationsJa(),
    AppLocalizationsDe(),
  ];
  final cases = <(String, String Function(AppLocalizations))>[
    ('campus.upstreamUnavailable', (l) => l.campusUpstreamUnavailable),
    ('campus.authorizationRequired', (l) => l.campusAuthRequired),
    ('campus.messageAuthorizationRequired', (l) => l.campusAuthRequired),
    ('campus.authorizationExpired', (l) => l.campusAuthExpired),
    ('campus.identityUnavailable', (l) => l.campusIdentityConflict),
    ('campus.connectionChanged', (l) => l.campusIdentityConflict),
    ('campus.messageUnavailable', (l) => l.campusMessageUnavailable),
    ('campus.calendarIncomplete', (l) => l.campusCalendarIncomplete),
    ('campus.rulesUnavailable', (l) => l.campusRulesUnavailable),
    ('campus.rulesInvalid', (l) => l.campusRulesUnavailable),
    ('campus.calendarEmpty', (l) => l.campusCalendarEmpty),
    ('campus.disabled', (l) => l.campusDisabled),
    ('campus.rulesAIUnavailable', (l) => l.campusRulesAIUnavailable),
    ('campus.rulesAIOutput', (l) => l.campusRulesAIOutput),
    ('campus.rulesLimited', (l) => l.campusRulesLimited),
    ('permission.emailRequired', (l) => l.authActivationRequired),
  ];
  for (final locale in locales) {
    for (final (code, text) in cases) {
      test('${locale.localeName} maps $code to dedicated copy', () {
        expect(campusError(locale, _api(code)), text(locale));
      });
    }
    test('${locale.localeName} keeps the generic fallback for unknown codes', () {
      expect(
        campusError(locale, _api('campus.unlistedCode')),
        locale.campusUnavailable,
      );
      expect(
        campusError(locale, 'transport failure'),
        locale.campusUnavailable,
      );
    });
  }
}
