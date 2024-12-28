from burp import IBurpExtender, IHttpListener, IContextMenuFactory
from java.util import ArrayList
from javax.swing import JMenuItem
import json
import urllib2

GEMINI_API = "GEMINI_API_KEY"

class BurpExtender(IBurpExtender, IHttpListener, IContextMenuFactory):
    def registerExtenderCallbacks(self, callbacks):
        self._callbacks = callbacks
        self._helpers = callbacks.getHelpers()
        callbacks.setExtensionName("Gemini Analyzer")
        callbacks.registerHttpListener(self)
        callbacks.registerContextMenuFactory(self)

        self._callbacks.printOutput("[*] Gemini AI Plugin loaded successfully")

        self._triggered_manually = False

    def createMenuItems(self, invocation):
        menu_list = ArrayList()
        menu_item = JMenuItem("Gemini-1.5 Analyze", actionPerformed=lambda x: self.trigger_extension(invocation))
        menu_list.add(menu_item)
        return menu_list

    def trigger_extension(self, invocation):
        self._triggered_manually = True

        selected_messages = invocation.getSelectedMessages()
        for message in selected_messages:
            self.processHttpMessage(invocation.getToolFlag(), False, message)

        self._triggered_manually = False

    def processHttpMessage(self, toolFlag, messageIsRequest, messageInfo):
        if not self._triggered_manually or messageIsRequest:
            return

        try:
            request = messageInfo.getRequest()
            service = messageInfo.getHttpService()
            analyzedRequest = self._helpers.analyzeRequest(service, request)

            url = str(analyzedRequest.getUrl())

            response = messageInfo.getResponse()
            if response:
                analyzedResponse = self._helpers.analyzeResponse(response)
                response_body = response[analyzedResponse.getBodyOffset():].tostring()

                analysis_result = self.send_to_ai_studio(url, response_body)

                if analysis_result:
                    self._callbacks.printOutput("[*] AI Studio Analysis Result: \n{}".format(analysis_result))
                else:
                    self._callbacks.printOutput("[-] No result returned from AI Studio.")
        except Exception as e:
            self._callbacks.printError("[-] Error processing HTTP message: {}".format(str(e)))


    def send_to_ai_studio(self, url, content):
        try:
            url_api = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + GEMINI_API
            headers = {"Content-Type": "application/json"}
            # lowkey this prompt is very important so adjust!!!
            prompt = (
                "Analyze the following HTTP response content for OWASP vulnerabilities. "
                "Look for XSS, SSRF, etc and give payloads [with the given url] to assist in identifying vulnerabilities."
                "Be serious and truly look at the content for web app vulns for bug bounties."
                "Make sure you stay within the scope of the URL of the ACTUAL response host, nothing more nothing less."
                "Here is the HTTP response content for the URL {}:\n\n{}".format(url, content)
            )

            payload = json.dumps({
                "contents": [{"parts": [{"text": prompt}]}]
            })

            self._callbacks.printOutput("[*] Sending request to AI Studio for analysis...")
            request = urllib2.Request(url_api, data=payload, headers=headers)
            response = urllib2.urlopen(request)

            response_data = response.read()

            result_text = json.loads(response_data).get("candidates", [{}])[0].get("content",{}).get("parts", [{}])[0].get("text","[-] No vulnerabilities detected.")
            return result_text
        except urllib2.HTTPError as e:
            self._callbacks.printError("[-] HTTPError: {} - {}".format(e.code, e.reason))
            return None
        except urllib2.URLError as e:
            self._callbacks.printError("[-] URLError: {}".format(e.reason))
            return None
        except Exception as e: # incase i miss some bs
            self._callbacks.printError("[-] Unexpected error: {}".format(str(e)))
            return None
