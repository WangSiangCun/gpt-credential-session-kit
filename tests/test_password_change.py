from types import SimpleNamespace
import pytest
from credential_session_kit.errors import CredentialSessionError
from credential_session_kit.engine.password_change import submit_password, validate_new_password

class Flow:
    _last_sentinel_token = 'security-token'
    result = SimpleNamespace(device_id='device')
    def __init__(self, status=200, result=None):
        self.calls=[]
        self.session=SimpleNamespace(request=lambda *a,**k: self.request(status,result,a,k))
    def request(self,status,result,args,kwargs):
        self.calls.append((args,kwargs))
        if isinstance(result, Exception):raise result
        return SimpleNamespace(status_code=status,json=lambda:result)
    def get_sentinel_token(self,device,**kwargs):assert kwargs=={'flow_name':'password_reset'}
    def _common_headers(self,referer):return {'Referer':referer}
    def _extract_page_type(self,result):return result.get('page',{}).get('type')
    def _extract_continue_url_from_step(self,result):return result.get('continue_url','')
    def _normalize_continue_url(self,url):return url

STEP={'page':{'type':'reset_password_new_password'},'continue_url':'https://auth.openai.com/reset-password/new-password'}

def test_only_explicit_reset_page_allows_mutation():
    f=Flow()
    assert submit_password(f,{'page':{'type':'email_otp_verification'}},'new-password-long',lambda x:None) is False
    assert not f.calls

def test_wrong_host_stops_before_post():
    f=Flow()
    with pytest.raises(CredentialSessionError):submit_password(f,{**STEP,'continue_url':'https://evil.test/reset-password/new-password'},'new-password-long',lambda x:None)
    assert not f.calls

def test_success_requires_confirmed_page():
    f=Flow(result={'page':{'type':'reset_password_success'}});stages=[]
    assert submit_password(f,STEP,'new-password-long',stages.append)
    assert stages[-1]=='security_password_applied'
    assert len(f.calls)==1 and f.calls[0][1]['allow_redirects'] is False

@pytest.mark.parametrize('status,result,code',[(200,{},'security_password_uncertain'),
    (200,TimeoutError(),'security_password_uncertain'),(403,{},'security_password_rejected'),
    (500,{},'security_password_uncertain')])
def test_no_retry_or_false_success(status,result,code):
    f=Flow(status,result);stages=[]
    with pytest.raises(CredentialSessionError) as error:submit_password(f,STEP,'new-password-long',stages.append)
    assert error.value.code==code
    assert len(f.calls)==1 and 'security_password_applied' not in stages

def test_short_password_blocked():
    with pytest.raises(CredentialSessionError):validate_new_password('short')
