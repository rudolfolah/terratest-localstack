FROM public.ecr.aws/amazonlinux/amazonlinux:2023-minimal

ARG RPM_ARCH
ADD https://downloads.aws.com/aws-emulator/packages/aws-emulator/LTS/aws-emulator-1.0.556-1-${RPM_ARCH}.rpm /tmp/aws-emulator.rpm
RUN rpm -i /tmp/aws-emulator.rpm && rm /tmp/aws-emulator.rpm && mkdir -p /home/emulator && chown 1000 /home/emulator

USER 1000
ENV HOME=/home/emulator EMULATOR_BIND_ADDRESS=0.0.0.0
ENTRYPOINT ["aws-emulator"]
