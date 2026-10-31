"""Allowlisted mobile build configuration; publishers never accept arbitrary hosts."""
import os
import re

# Dev previews install beside the stable app; Gradle derives the same identity
# from the YOURTJ_ENVIRONMENT define (android/app/build.gradle.kts).
ANDROID_PACKAGES = {'dev': 'tj.yourtj.forum_app.dev', 'production': 'tj.yourtj.forum_app'}


def android_package(environment):
    if environment not in ANDROID_PACKAGES:
        raise ValueError('Unknown mobile environment')
    return ANDROID_PACKAGES[environment]


def dart_defines(environment=None, source=None, preview=None):
    environment = environment or os.environ.get('MOBILE_ENVIRONMENT', 'production')
    source = source if source is not None else os.environ.get('SOURCE_SHA', '')
    preview = preview if preview is not None else os.environ.get('MOBILE_PREVIEW') == 'true'
    if environment not in {'dev', 'production'}:
        raise ValueError('Unknown mobile environment')
    if not re.fullmatch(r'[0-9a-f]{40}', source):
        raise ValueError('Mobile builds require a full source SHA')
    if environment == 'dev' and not preview:
        raise ValueError('A dev build may only use preview distribution')
    host = 'dev.yourtj.de' if environment == 'dev' else 'f.yourtj.de'
    values = {'YOURTJ_API_BASE_URL': f'https://{host}', 'YOURTJ_OIDC_ISSUER': f'https://{host}/api/oauth',
              'YOURTJ_OIDC_CLIENT_ID': 'yourtj-mobile', 'YOURTJ_ENVIRONMENT': environment,
              'YOURTJ_SOURCE_SHA': source, 'YOURTJ_PREVIEW': str(preview).lower()}
    return [f'--dart-define={key}={value}' for key, value in values.items()]


if __name__ == '__main__':
    print('\n'.join(dart_defines()))
