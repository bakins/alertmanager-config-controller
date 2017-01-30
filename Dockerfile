FROM scratch
MAINTAINER Brian Akins <brian@akins.org>
COPY alertmanager-config-controller.linux /alertmanager-config-controller
ENTRYPOINT [ "/alertmanager-config-controller" ]
