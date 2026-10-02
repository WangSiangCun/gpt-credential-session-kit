from types import SimpleNamespace

import pytest
from credential_session_kit.mail_code import MailCodeProvider, extract_message, validate_mail_code_url
from credential_session_kit.errors import CredentialSessionError

URL = 'https://gapi.mailsapi.com/api/code/fetch?token=fixture-token&uid=fixture-uid'


@pytest.mark.parametrize('url', [
    'http://gapi.mailsapi.com/api/code/fetch?token=a&uid=b',
    'https://localhost/api/code/fetch?token=a&uid=b',
    'https://gapi.mailsapi.com.evil.test/api/code/fetch?token=a&uid=b',
    'https://gapi.mailsapi.com@evil.test/api/code/fetch?token=a&uid=b',
    URL + '&token=duplicate', URL + '#fragment', URL + '&callback=https://localhost',
    'https://gapi.mailsapi.com/elsewhere?token=a&uid=b',
])
def test_reject_unapproved_url_without_revealing_secrets(url):
    with pytest.raises(CredentialSessionError) as error:
        validate_mail_code_url(url)
    assert error.value.code == 'mail_code_url'
    assert 'fixture-token' not in str(error.value)


def test_status_code_is_not_an_otp():
    assert extract_message({'code':200, 'data':None}) is None
    with pytest.raises(CredentialSessionError):
        extract_message({'code':123456, 'data':None})


class Session:
    def __init__(self, payloads): self.payloads = iter(payloads); self.calls=[]
    def get(self, *args, **kwargs):
        self.calls.append(kwargs)
        p = next(self.payloads)
        return SimpleNamespace(status_code=200, content=b'{}', json=lambda:p)


def test_skip_baseline_and_consume_only_new_code():
    s = Session([{'code':200, 'data':'123456'}, {'code':200, 'data':'123456'}, {'code':200, 'data':'654321'}])
    clock=[0]
    p=MailCodeProvider('a@example.com',URL,s,clock=lambda:clock[0],sleep=lambda n:clock.__setitem__(0,clock[0]+n))
    p.prepare()
    assert p.wait_for_otp('a@example.com') == '654321'
    assert all(x['allow_redirects'] is False for x in s.calls)
    assert all(x['headers'] == {'Accept':'application/json'} for x in s.calls)


def test_old_timestamp_never_accepted_even_if_code_different():
    s=Session([{'code':200,'data':None},
               {'code':200,'data':{'code':'123456','timestamp':10}},
               {'code':200,'data':{'code':'654321','timestamp':100}}])
    clock=[0]
    p=MailCodeProvider('a@example.com',URL,s,clock=lambda:clock[0],wall_clock=lambda:100,
                      sleep=lambda n:clock.__setitem__(0,clock[0]+n))
    p.prepare()
    assert p.wait_for_otp('a@example.com',issued_after=90) == '654321'


def test_identity_and_unprepared_provider_block_polling():
    p=MailCodeProvider('a@example.com',URL,Session([]))
    with pytest.raises(CredentialSessionError): p.wait_for_otp('a@example.com')
    p.prepared=True
    with pytest.raises(CredentialSessionError): p.wait_for_otp('b@example.com')


@pytest.mark.parametrize('data', [{'code':654321}, {'message':'Your code is 123456'},
                                {'code':'123456','timestamp':'yesterday'}, ['123456']])
def test_no_heuristic_parsing_unknown_payloads(data):
    with pytest.raises(CredentialSessionError): extract_message({'code':200,'data':data})


def test_timeout_is_bounded():
    clock=[0]
    p=MailCodeProvider('a@example.com',URL,Session([{'code':200,'data':None}]*5),
                      clock=lambda:clock[0],sleep=lambda n:clock.__setitem__(0,clock[0]+n))
    p.prepare()
    with pytest.raises(CredentialSessionError) as error:p.wait_for_otp('a@example.com',timeout=6)
    assert error.value.code=='mail_code_timeout'
    assert clock[0]==6
