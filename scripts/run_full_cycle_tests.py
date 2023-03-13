#!/usr/bin/env python3
# -*- coding: utf-8 -*-

#
# Copyright (c) 2021 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

"""
This script runs the integration full cycle tests.
"""

import os
import sys

import build

# Addresses for the build failure message:
FAILURE_FROM = "Jenkins <noreply@sdev.devshift.net>"
FAILURE_TO = ", ".join([
    "Developers <ocm-devel@redhat.com>",
])

# Template used to generate the subject of the build failure message:
FAILURE_SUBJECT = """
[Jenkins] Terraform provider ocm full cycle tests job {{ BUILD_NUMBER }} has failed
"""

# Template used to generate the body of the build failure message:
FAILURE_BODY = """
Hi,

The terraform provider OCM full cycle tests job {{ BUILD_NUMBER }} has failed.

You can find the details here:

{{ BUILD_URL }}

Regards,
Jenkins
"""


def getenv(name, default=None):
    """
    Gets the value of an environment variable, and assigns a default value
    if it isn't defined. If the variable doesn't have a value and no default
    is provided then an exception is raised.
    """
    value = os.getenv(name)
    if value is None:
        if default is None:
            raise Exception(f"Environment variable '{name}' is mandatory")
        value = default
    return value


def main():
    """
    Main function.
    """

    # Create the logger:
    log = build.Logger()

    # Get the location of the API server:
    test_gateway_url=getenv("TEST_GATEWAY_URL")
    log.redact(test_gateway_url)

    # Get the tokens used for the tests:
    test_token=getenv("TEST_OFFLINE_TOKEN")
    log.redact(test_token)

    # Get the SMTP connection details:
    smtp_server = getenv("SMTP_SERVER", default="")
    smtp_port = getenv("SMTP_PORT", default="25")
    smtp_user = os.getenv("SMTP_USER", default="")
    smtp_password = os.getenv("SMTP_PASSWORD", default="")
    log.info(smtp_server)
    log.info(smtp_port)
    log.info(smtp_user)
    log.redact(smtp_password)

    # Run the tests:
    log.info("Running integration full cycle tests")
    make = build.Make(log=log)
    result = make.run(
        variables={
            "test_gateway_url": test_gateway_url,
            "test_offline_token": test_token,
        },
        targets={
            "e2e_test",
        },
    )
    if result != 0:
        log.info(f"Full cycle tests failed with exit code {result}")
    else:
        log.info("Full cycle tests succeeded")

    # If the tests have failed and mail notifications are enabled, then prepare
    # and send the error report:
    if result != 0 and smtp_server != "":
        log.info("Sending e-mail notification")
        mailer = build.Mailer(
            log=log,
            server=smtp_server,
            port=smtp_port,
            user=smtp_user,
            password=smtp_password,
        )
        mailer.send(
            sender=FAILURE_FROM,
            receiver=FAILURE_TO,
            data=os.environ,
            subject=FAILURE_SUBJECT,
            body=FAILURE_BODY,
        )

    # Return the result of the tests:
    sys.exit(result)


if __name__ == "__main__":
    main()
